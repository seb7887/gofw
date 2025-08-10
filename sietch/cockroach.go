package sietch

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"reflect"
	"strings"
)

type CockroachDBConnector[T any, ID comparable] struct {
	pool      *pgxpool.Pool
	tableName string
	getID     func(*T) ID
	columns   []string
}

func NewCockroachDBConnPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, dsn)
}

func sanitizeIdentifier(name string) error {
	if name == "" {
		return fmt.Errorf("identifier cannot be empty")
	}
	// Solo permitir letras, números, guiones bajos
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_') {
			return fmt.Errorf("invalid character in identifier: %c", r)
		}
	}
	return nil
}

func quoteIdentifier(name string) string {
	return `"` + name + `"`
}

// NewCockroachDBConnector CockroachDB implementation of Repository interface
func NewCockroachDBConnector[T any, ID comparable](pool *pgxpool.Pool, tableName string, getID func(*T) ID) (*CockroachDBConnector[T, ID], error) {
	if pool == nil {
		return nil, fmt.Errorf("pool cannot be nil")
	}
	if getID == nil {
		return nil, fmt.Errorf("getID function cannot be nil")
	}
	if err := sanitizeIdentifier(tableName); err != nil {
		return nil, fmt.Errorf("invalid table name: %w", err)
	}
	
	columns, err := getColumns[T]()
	if err != nil {
		return nil, err
	}
	
	// Validar nombres de columnas
	for _, col := range columns {
		if err := sanitizeIdentifier(col); err != nil {
			return nil, fmt.Errorf("invalid column name '%s': %w", col, err)
		}
	}

	return &CockroachDBConnector[T, ID]{
		pool:      pool,
		tableName: tableName,
		getID:     getID,
		columns:   columns,
	}, nil
}

func getColumns[T any]() ([]string, error) {
	var t T
	typ := reflect.TypeOf(t)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("columns must be a struct")
	}

	var columns []string
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("db")
		if tag != "" {
			columns = append(columns, tag)
		}
	}

	if len(columns) == 0 {
		return nil, fmt.Errorf("no columns found")
	}

	return columns, nil
}

func joinColumns(columns []string) string {
	return strings.Join(columns, ", ")
}

func joinQuotedColumns(columns []string) string {
	quoted := make([]string, len(columns))
	for i, col := range columns {
		quoted[i] = quoteIdentifier(col)
	}
	return strings.Join(quoted, ", ")
}

func buildPlaceholders(n int) string {
	placeholders := make([]string, n)
	for i := 0; i < n; i++ {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	return strings.Join(placeholders, ", ")
}

func (r *CockroachDBConnector[T, ID]) getValues(item *T) ([]any, error) {
	v := reflect.ValueOf(item)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("item must be a struct")
	}
	typ := v.Type()
	var values []any
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("db")
		if tag != "" {
			values = append(values, v.Field(i).Interface())
		}
	}
	if len(values) != len(r.columns) {
		return nil, fmt.Errorf("number of values does not match the number of columns")
	}

	return values, nil
}

func (r *CockroachDBConnector[T, ID]) getScanDestinations(ptr *T) ([]any, error) {
	v := reflect.ValueOf(ptr).Elem()
	typ := v.Type()
	var dests []any
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("db")
		if tag != "" {
			dests = append(dests, v.Field(i).Addr().Interface())
		}
	}
	if len(dests) != len(r.columns) {
		return nil, fmt.Errorf("number of values does not match the number of columns")
	}
	return dests, nil
}

func (r *CockroachDBConnector[T, ID]) Create(ctx context.Context, item *T) error {
	values, err := r.getValues(item)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		quoteIdentifier(r.tableName),
		joinQuotedColumns(r.columns),
		buildPlaceholders(len(r.columns)),
	)
	_, err = r.pool.Exec(ctx, query, values...)
	return err
}

func (r *CockroachDBConnector[T, ID]) Get(ctx context.Context, id ID) (*T, error) {
	var t T
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1",
		joinQuotedColumns(r.columns),
		quoteIdentifier(r.tableName),
		quoteIdentifier(r.columns[0]),
	)
	row := r.pool.QueryRow(ctx, query, id)
	dests, err := r.getScanDestinations(&t)
	if err != nil {
		return nil, err
	}

	err = row.Scan(dests...)

	return &t, err
}

func (r *CockroachDBConnector[T, ID]) BatchCreate(ctx context.Context, items []T) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				// TODO: Log rollback error: rollbackErr
			}
		} else {
			if commitErr := tx.Commit(ctx); commitErr != nil {
				// TODO: Log commit error: commitErr
				err = commitErr // Set error so it gets returned
			}
		}
	}()

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		quoteIdentifier(r.tableName),
		joinQuotedColumns(r.columns),
		buildPlaceholders(len(r.columns)),
	)

	for _, item := range items {
		values, err := r.getValues(&item)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, query, values...)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *CockroachDBConnector[T, ID]) Query(ctx context.Context, filter *Filter) ([]T, error) {
	if filter == nil {
		return nil, fmt.Errorf("filter cannot be nil")
	}
	query, args := r.queryBuilder(filter)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		var item T
		dests, err := r.getScanDestinations(&item)
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

func (r *CockroachDBConnector[T, ID]) Update(ctx context.Context, item *T) error {
	values, err := r.getValues(item)
	if err != nil {
		return err
	}

	var setClause []string
	numCols := len(r.columns)
	for i := 1; i < numCols; i++ {
		setClause = append(setClause, fmt.Sprintf("%s = $%d", quoteIdentifier(r.columns[i]), i))
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = $%d",
		quoteIdentifier(r.tableName),
		strings.Join(setClause, ", "),
		quoteIdentifier(r.columns[0]),
		numCols,
	)

	id := r.getID(item)
	args := append(values[1:], id)
	ct, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	if ct.RowsAffected() == 0 {
		return ErrNoUpdateItem
	}

	return nil
}

func (r *CockroachDBConnector[T, ID]) BatchUpdate(ctx context.Context, items []T) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				// TODO: Log rollback error: rollbackErr
			}
		} else {
			if commitErr := tx.Commit(ctx); commitErr != nil {
				// TODO: Log commit error: commitErr
				err = commitErr // Set error so it gets returned
			}
		}
	}()

	numCols := len(r.columns)
	var setClauses []string
	for i := 1; i < numCols; i++ {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", quoteIdentifier(r.columns[i]), i))
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = $%d",
		quoteIdentifier(r.tableName),
		strings.Join(setClauses, ", "),
		quoteIdentifier(r.columns[0]),
		numCols,
	)

	_, err = tx.Prepare(ctx, "batch_update_stmt", query)
	if err != nil {
		return err
	}

	for _, item := range items {
		values, err := r.getValues(&item)
		if err != nil {
			return err
		}

		id := r.getID(&item)
		args := append(values[1:], id)
		ct, err := tx.Exec(ctx, "batch_update_stmt", args...)
		if err != nil {
			return err
		}

		if ct.RowsAffected() == 0 {
			return fmt.Errorf("batch update item %v does not exist", item)
		}
	}

	return nil
}

func (r *CockroachDBConnector[T, ID]) Delete(ctx context.Context, id ID) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = $1",
		quoteIdentifier(r.tableName),
		quoteIdentifier(r.columns[0]),
	)

	ct, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if ct.RowsAffected() == 0 {
		return ErrNoDeleteItem
	}

	return nil
}

func (r *CockroachDBConnector[T, ID]) BatchDelete(ctx context.Context, items []ID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				// TODO: Log rollback error: rollbackErr
			}
		} else {
			if commitErr := tx.Commit(ctx); commitErr != nil {
				// TODO: Log commit error: commitErr
				err = commitErr // Set error so it gets returned
			}
		}
	}()

	query := fmt.Sprintf("DELETE FROM %s WHERE %s = $1",
		quoteIdentifier(r.tableName),
		quoteIdentifier(r.columns[0]),
	)
	_, err = tx.Prepare(ctx, "batch_delete_stmt", query)
	if err != nil {
		return err
	}

	for _, id := range items {
		ct, err := tx.Exec(ctx, "batch_delete_stmt", id)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return fmt.Errorf("%v row not deleted", id)
		}
	}

	return nil
}

func (r *CockroachDBConnector[T, ID]) queryBuilder(filter *Filter) (string, []any) {
	whereClause := ""
	var args []any
	
	if filter == nil || len(filter.Conditions) == 0 {
		// Sin condiciones, devolver query simple
		query := fmt.Sprintf("SELECT %s FROM %s",
			joinQuotedColumns(r.columns),
			quoteIdentifier(r.tableName),
		)
		return query, args
	}
	
	for i, condition := range filter.Conditions {
		if i == 0 {
			whereClause = "WHERE "
		} else {
			whereClause += " AND "
		}
		whereClause += fmt.Sprintf("%s %s $%d", quoteIdentifier(condition.Field), condition.Operator, i+1)
		args = append(args, condition.Value)
	}

	query := fmt.Sprintf("SELECT %s FROM %s %s",
		joinQuotedColumns(r.columns),
		quoteIdentifier(r.tableName),
		whereClause,
	)

	return query, args
}
