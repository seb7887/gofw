package sietch

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Queryable interface abstracts both pgxpool.Pool and pgx.Tx
type Queryable interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// cockroachDBTx wraps a CockroachDBConnector to use a transaction instead of the pool
type cockroachDBTx[T any, ID comparable] struct {
	connector *CockroachDBConnector[T, ID]
	tx        pgx.Tx
	ctx       context.Context
}

// WithTx executes the given function within a transaction.
// If the function returns an error, the transaction is rolled back.
// If the function returns nil, the transaction is committed.
// If the function panics, the transaction is rolled back and the panic is re-raised.
func (r *CockroachDBConnector[T, ID]) WithTx(ctx context.Context, fn TxFunc[T, ID]) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create transaction-scoped repository
	txRepo := &cockroachDBTx[T, ID]{
		connector: r,
		tx:        tx,
		ctx:       ctx,
	}

	// Defer rollback in case of panic or error
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	// Execute the user function
	err = fn(txRepo)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx error: %w, rollback error: %v", err, rbErr)
		}
		return err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Implement Repository interface for cockroachDBTx
// All methods delegate to the connector but use tx instead of pool

func (t *cockroachDBTx[T, ID]) Create(ctx context.Context, item *T) error {
	if item == nil {
		return fmt.Errorf("item cannot be nil")
	}

	values, err := t.connector.getValues(item)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		quoteIdentifier(t.connector.tableName),
		joinQuotedColumns(t.connector.columns),
		buildPlaceholders(len(t.connector.columns)),
	)
	_, err = t.tx.Exec(ctx, query, values...)

	// Check for duplicate key error
	if err != nil && contains(err.Error(), "duplicate key") {
		return ErrItemAlreadyExists
	}

	return err
}

func (t *cockroachDBTx[T, ID]) Get(ctx context.Context, id ID) (*T, error) {
	var item T
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1",
		joinQuotedColumns(t.connector.columns),
		quoteIdentifier(t.connector.tableName),
		quoteIdentifier(t.connector.columns[0]),
	)
	row := t.tx.QueryRow(ctx, query, id)
	dests, err := t.connector.getScanDestinations(&item)
	if err != nil {
		return nil, err
	}

	err = row.Scan(dests...)
	return &item, err
}

func (t *cockroachDBTx[T, ID]) BatchCreate(ctx context.Context, items []T) error {
	if len(items) == 0 {
		return nil
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		quoteIdentifier(t.connector.tableName),
		joinQuotedColumns(t.connector.columns),
		buildPlaceholders(len(t.connector.columns)),
	)

	for _, item := range items {
		values, err := t.connector.getValues(&item)
		if err != nil {
			return err
		}
		_, err = t.tx.Exec(ctx, query, values...)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *cockroachDBTx[T, ID]) Query(ctx context.Context, filter *Filter) ([]T, error) {
	if filter == nil {
		return nil, fmt.Errorf("filter cannot be nil")
	}
	query, args, err := t.connector.queryBuilder(filter)
	if err != nil {
		return nil, err
	}
	rows, err := t.tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		var item T
		dests, err := t.connector.getScanDestinations(&item)
		if err != nil {
			return nil, err
		}
		if err := rows.Scan(dests...); err != nil {
			return nil, err
		}
		results = append(results, item)
	}

	return results, rows.Err()
}

func (t *cockroachDBTx[T, ID]) Update(ctx context.Context, item *T) error {
	if item == nil {
		return fmt.Errorf("item cannot be nil")
	}

	values, err := t.connector.getValues(item)
	if err != nil {
		return err
	}

	var setClause []string
	numCols := len(t.connector.columns)
	for i := 1; i < numCols; i++ {
		setClause = append(setClause, fmt.Sprintf("%s = $%d", quoteIdentifier(t.connector.columns[i]), i))
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = $%d",
		quoteIdentifier(t.connector.tableName),
		joinString(setClause, ", "),
		quoteIdentifier(t.connector.columns[0]),
		numCols,
	)

	id := t.connector.getID(item)
	args := append(values[1:], id)
	ct, err := t.tx.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	if ct.RowsAffected() == 0 {
		return ErrNoUpdateItem
	}

	return nil
}

func (t *cockroachDBTx[T, ID]) BatchUpdate(ctx context.Context, items []T) error {
	if len(items) == 0 {
		return nil
	}

	numCols := len(t.connector.columns)
	var setClauses []string
	for i := 1; i < numCols; i++ {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", quoteIdentifier(t.connector.columns[i]), i))
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = $%d",
		quoteIdentifier(t.connector.tableName),
		joinString(setClauses, ", "),
		quoteIdentifier(t.connector.columns[0]),
		numCols,
	)

	_, err := t.tx.Prepare(ctx, "tx_batch_update_stmt", query)
	if err != nil {
		return err
	}

	for _, item := range items {
		values, err := t.connector.getValues(&item)
		if err != nil {
			return err
		}

		id := t.connector.getID(&item)
		args := append(values[1:], id)
		ct, err := t.tx.Exec(ctx, "tx_batch_update_stmt", args...)
		if err != nil {
			return err
		}

		if ct.RowsAffected() == 0 {
			return fmt.Errorf("batch update item %v does not exist", item)
		}
	}

	return nil
}

func (t *cockroachDBTx[T, ID]) Delete(ctx context.Context, id ID) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = $1",
		quoteIdentifier(t.connector.tableName),
		quoteIdentifier(t.connector.columns[0]),
	)

	ct, err := t.tx.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if ct.RowsAffected() == 0 {
		return ErrNoDeleteItem
	}

	return nil
}

func (t *cockroachDBTx[T, ID]) BatchDelete(ctx context.Context, items []ID) error {
	if len(items) == 0 {
		return nil
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s = $1",
		quoteIdentifier(t.connector.tableName),
		quoteIdentifier(t.connector.columns[0]),
	)
	_, err := t.tx.Prepare(ctx, "tx_batch_delete_stmt", query)
	if err != nil {
		return err
	}

	for _, id := range items {
		ct, err := t.tx.Exec(ctx, "tx_batch_delete_stmt", id)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return fmt.Errorf("%v row not deleted", id)
		}
	}

	return nil
}

func (t *cockroachDBTx[T, ID]) Count(ctx context.Context, filter *Filter) (int64, error) {
	if filter == nil {
		return 0, fmt.Errorf("filter cannot be nil")
	}

	var args []any
	argIndex := 1

	query := "SELECT COUNT(*) FROM " + quoteIdentifier(t.connector.tableName)

	// Build WHERE clause
	if len(filter.Conditions) > 0 {
		whereClause, whereArgs, err := t.connector.buildWhereClause(filter.Conditions, &argIndex)
		if err != nil {
			return 0, err
		}
		query += " WHERE " + whereClause
		args = append(args, whereArgs...)
	}

	var count int64
	err := t.tx.QueryRow(ctx, query, args...).Scan(&count)
	return count, err
}

// Exists checks if an entity with the given ID exists within the transaction
func (t *cockroachDBTx[T, ID]) Exists(ctx context.Context, id ID) (bool, error) {
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE %s = $1)",
		quoteIdentifier(t.connector.tableName),
		quoteIdentifier(t.connector.columns[0]),
	)

	var exists bool
	err := t.tx.QueryRow(ctx, query, id).Scan(&exists)
	return exists, err
}

// Upsert creates a new entity or updates an existing one within the transaction
func (t *cockroachDBTx[T, ID]) Upsert(ctx context.Context, item *T) error {
	if item == nil {
		return fmt.Errorf("item cannot be nil")
	}

	values, err := t.connector.getValues(item)
	if err != nil {
		return err
	}

	// Build the SET clause for ON CONFLICT DO UPDATE
	var setClauses []string
	numCols := len(t.connector.columns)
	for i := 1; i < numCols; i++ {
		setClauses = append(setClauses, fmt.Sprintf("%s = EXCLUDED.%s",
			quoteIdentifier(t.connector.columns[i]),
			quoteIdentifier(t.connector.columns[i]),
		))
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s",
		quoteIdentifier(t.connector.tableName),
		joinQuotedColumns(t.connector.columns),
		buildPlaceholders(len(t.connector.columns)),
		quoteIdentifier(t.connector.columns[0]),
		joinString(setClauses, ", "),
	)

	_, err = t.tx.Exec(ctx, query, values...)
	return err
}

// BatchUpsert creates or updates multiple entities within the transaction
func (t *cockroachDBTx[T, ID]) BatchUpsert(ctx context.Context, items []T) error {
	if len(items) == 0 {
		return nil
	}

	// Build the SET clause for ON CONFLICT DO UPDATE
	var setClauses []string
	numCols := len(t.connector.columns)
	for i := 1; i < numCols; i++ {
		setClauses = append(setClauses, fmt.Sprintf("%s = EXCLUDED.%s",
			quoteIdentifier(t.connector.columns[i]),
			quoteIdentifier(t.connector.columns[i]),
		))
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s",
		quoteIdentifier(t.connector.tableName),
		joinQuotedColumns(t.connector.columns),
		buildPlaceholders(len(t.connector.columns)),
		quoteIdentifier(t.connector.columns[0]),
		joinString(setClauses, ", "),
	)

	for _, item := range items {
		values, err := t.connector.getValues(&item)
		if err != nil {
			return err
		}
		_, err = t.tx.Exec(ctx, query, values...)
		if err != nil {
			return err
		}
	}

	return nil
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func joinString(slice []string, sep string) string {
	if len(slice) == 0 {
		return ""
	}
	result := slice[0]
	for i := 1; i < len(slice); i++ {
		result += sep + slice[i]
	}
	return result
}
