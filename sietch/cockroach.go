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
	if item == nil {
		return fmt.Errorf("item cannot be nil")
	}

	values, err := r.getValues(item)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		quoteIdentifier(r.tableName),
		joinQuotedColumns(r.columns),
		buildPlaceholders(len(r.columns)),
	)

	queryable := r.getQueryable(ctx)
	_, err = queryable.Exec(ctx, query, values...)

	// Check for duplicate key error
	if err != nil && strings.Contains(err.Error(), "duplicate key") {
		return ErrItemAlreadyExists
	}

	return err
}

func (r *CockroachDBConnector[T, ID]) Get(ctx context.Context, id ID) (*T, error) {
	var t T
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1",
		joinQuotedColumns(r.columns),
		quoteIdentifier(r.tableName),
		quoteIdentifier(r.columns[0]),
	)

	queryable := r.getQueryable(ctx)
	row := queryable.QueryRow(ctx, query, id)
	dests, err := r.getScanDestinations(&t)
	if err != nil {
		return nil, err
	}

	err = row.Scan(dests...)

	return &t, err
}

func (r *CockroachDBConnector[T, ID]) BatchCreate(ctx context.Context, items []T) error {
	if len(items) == 0 {
		return nil
	}

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
	query, args, err := r.queryBuilder(filter)
	if err != nil {
		return nil, err
	}

	queryable := r.getQueryable(ctx)
	rows, err := queryable.Query(ctx, query, args...)
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

// Count returns the number of items matching the filter
func (r *CockroachDBConnector[T, ID]) Count(ctx context.Context, filter *Filter) (int64, error) {
	if filter == nil {
		return 0, fmt.Errorf("filter cannot be nil")
	}

	var args []any
	argIndex := 1

	query := "SELECT COUNT(*) FROM " + quoteIdentifier(r.tableName)

	// Build WHERE clause
	if len(filter.Conditions) > 0 {
		whereClause, whereArgs, err := r.buildWhereClause(filter.Conditions, &argIndex)
		if err != nil {
			return 0, err
		}
		query += " WHERE " + whereClause
		args = append(args, whereArgs...)
	}

	queryable := r.getQueryable(ctx)
	var count int64
	err := queryable.QueryRow(ctx, query, args...).Scan(&count)
	return count, err
}

func (r *CockroachDBConnector[T, ID]) Update(ctx context.Context, item *T) error {
	if item == nil {
		return fmt.Errorf("item cannot be nil")
	}

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

	queryable := r.getQueryable(ctx)
	ct, err := queryable.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	if ct.RowsAffected() == 0 {
		return ErrNoUpdateItem
	}

	return nil
}

func (r *CockroachDBConnector[T, ID]) BatchUpdate(ctx context.Context, items []T) error {
	if len(items) == 0 {
		return nil
	}

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

	queryable := r.getQueryable(ctx)
	ct, err := queryable.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if ct.RowsAffected() == 0 {
		return ErrNoDeleteItem
	}

	return nil
}

func (r *CockroachDBConnector[T, ID]) BatchDelete(ctx context.Context, items []ID) error {
	if len(items) == 0 {
		return nil
	}

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

// validateFilterField checks if a field exists in the known columns
func (r *CockroachDBConnector[T, ID]) validateFilterField(field string) error {
	found := false
	for _, col := range r.columns {
		if col == field {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("unknown field '%s' for filtering", field)
	}

	return sanitizeIdentifier(field)
}

func (r *CockroachDBConnector[T, ID]) queryBuilder(filter *Filter) (string, []any, error) {
	var args []any
	argIndex := 1

	// Start with SELECT
	selectClause := "SELECT "
	if filter != nil && filter.Distinct {
		selectClause += "DISTINCT "
	}
	selectClause += joinQuotedColumns(r.columns)

	query := selectClause + " FROM " + quoteIdentifier(r.tableName)

	// Build WHERE clause
	if filter != nil && len(filter.Conditions) > 0 {
		whereClause, whereArgs, err := r.buildWhereClause(filter.Conditions, &argIndex)
		if err != nil {
			return "", nil, err
		}
		query += " WHERE " + whereClause
		args = append(args, whereArgs...)
	}

	// Build ORDER BY clause
	if filter != nil && len(filter.Sort) > 0 {
		orderByClause, err := r.buildOrderByClause(filter.Sort)
		if err != nil {
			return "", nil, err
		}
		query += " " + orderByClause
	}

	// Add LIMIT
	if filter != nil && filter.Limit != nil {
		query += fmt.Sprintf(" LIMIT %d", *filter.Limit)
	}

	// Add OFFSET
	if filter != nil && filter.Offset != nil {
		query += fmt.Sprintf(" OFFSET %d", *filter.Offset)
	}

	return query, args, nil
}

func (r *CockroachDBConnector[T, ID]) buildWhereClause(conditions []Condition, argIndex *int) (string, []any, error) {
	var clauses []string
	var args []any

	for _, condition := range conditions {
		clause, condArgs, err := r.buildConditionClause(condition, argIndex)
		if err != nil {
			return "", nil, err
		}

		clauses = append(clauses, clause)
		args = append(args, condArgs...)
	}

	return strings.Join(clauses, " AND "), args, nil
}

func (r *CockroachDBConnector[T, ID]) buildConditionClause(condition Condition, argIndex *int) (string, []any, error) {
	// Check if this is a composite condition (logical grouping)
	if condition.IsComposite() {
		return r.buildCompositeCondition(condition, argIndex)
	}

	// This is a leaf condition (field comparison)
	return r.buildLeafCondition(condition, argIndex)
}

func (r *CockroachDBConnector[T, ID]) buildLeafCondition(condition Condition, argIndex *int) (string, []any, error) {
	// Validate field
	if err := r.validateFilterField(condition.Field); err != nil {
		return "", nil, err
	}

	field := quoteIdentifier(condition.Field)
	var clause string
	var args []any

	switch condition.Operator {
	case OpEqual, OpNotEqual, OpGreaterThan, OpLessThan, OpGreaterThanOrEqual, OpLessThanOrEqual, OpLike, OpILike:
		clause = fmt.Sprintf("%s %s $%d", field, condition.Operator, *argIndex)
		args = append(args, condition.Value)
		*argIndex++

	case OpIn, OpNotIn:
		// Value should be a slice
		v := reflect.ValueOf(condition.Value)
		if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
			return "", nil, fmt.Errorf("IN/NOT IN operator requires slice value")
		}

		if v.Len() == 0 {
			return "", nil, fmt.Errorf("IN/NOT IN operator requires non-empty slice")
		}

		placeholders := make([]string, v.Len())
		for i := 0; i < v.Len(); i++ {
			placeholders[i] = fmt.Sprintf("$%d", *argIndex)
			args = append(args, v.Index(i).Interface())
			*argIndex++
		}
		clause = fmt.Sprintf("%s %s (%s)", field, condition.Operator, strings.Join(placeholders, ", "))

	case OpIsNull, OpIsNotNull:
		clause = fmt.Sprintf("%s %s", field, condition.Operator)

	case OpBetween:
		// Value should be a slice with 2 elements
		v := reflect.ValueOf(condition.Value)
		if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
			return "", nil, fmt.Errorf("BETWEEN operator requires slice value")
		}

		if v.Len() != 2 {
			return "", nil, fmt.Errorf("BETWEEN operator requires exactly 2 values")
		}

		clause = fmt.Sprintf("%s BETWEEN $%d AND $%d", field, *argIndex, *argIndex+1)
		args = append(args, v.Index(0).Interface(), v.Index(1).Interface())
		*argIndex += 2

	default:
		return "", nil, fmt.Errorf("unsupported operator: %s", condition.Operator)
	}

	return clause, args, nil
}

func (r *CockroachDBConnector[T, ID]) buildCompositeCondition(condition Condition, argIndex *int) (string, []any, error) {
	if len(condition.Conditions) == 0 {
		return "", nil, fmt.Errorf("composite condition must have nested conditions")
	}

	var clauses []string
	var args []any

	// Build all nested conditions
	for _, nested := range condition.Conditions {
		clause, nestedArgs, err := r.buildConditionClause(nested, argIndex)
		if err != nil {
			return "", nil, err
		}
		clauses = append(clauses, clause)
		args = append(args, nestedArgs...)
	}

	var result string
	switch condition.LogicalOp {
	case LogicalAND:
		result = "(" + strings.Join(clauses, " AND ") + ")"
	case LogicalOR:
		result = "(" + strings.Join(clauses, " OR ") + ")"
	case LogicalNOT:
		if len(clauses) != 1 {
			return "", nil, fmt.Errorf("NOT operator requires exactly one condition")
		}
		result = "NOT (" + clauses[0] + ")"
	default:
		return "", nil, fmt.Errorf("unsupported logical operator: %s", condition.LogicalOp)
	}

	return result, args, nil
}

func (r *CockroachDBConnector[T, ID]) buildOrderByClause(sortFields []SortField) (string, error) {
	var parts []string

	for _, sf := range sortFields {
		// Validate field
		if err := r.validateFilterField(sf.Field); err != nil {
			return "", err
		}

		parts = append(parts, fmt.Sprintf("%s %s", quoteIdentifier(sf.Field), sf.Direction))
	}

	return "ORDER BY " + strings.Join(parts, ", "), nil
}

// Exists checks if an entity with the given ID exists
func (r *CockroachDBConnector[T, ID]) Exists(ctx context.Context, id ID) (bool, error) {
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE %s = $1)",
		quoteIdentifier(r.tableName),
		quoteIdentifier(r.columns[0]),
	)

	queryable := r.getQueryable(ctx)
	var exists bool
	err := queryable.QueryRow(ctx, query, id).Scan(&exists)
	return exists, err
}

// Upsert creates a new entity or updates an existing one using ON CONFLICT
func (r *CockroachDBConnector[T, ID]) Upsert(ctx context.Context, item *T) error {
	if item == nil {
		return fmt.Errorf("item cannot be nil")
	}

	values, err := r.getValues(item)
	if err != nil {
		return err
	}

	// Build the SET clause for ON CONFLICT DO UPDATE
	var setClauses []string
	numCols := len(r.columns)
	for i := 1; i < numCols; i++ {
		setClauses = append(setClauses, fmt.Sprintf("%s = EXCLUDED.%s",
			quoteIdentifier(r.columns[i]),
			quoteIdentifier(r.columns[i]),
		))
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s",
		quoteIdentifier(r.tableName),
		joinQuotedColumns(r.columns),
		buildPlaceholders(len(r.columns)),
		quoteIdentifier(r.columns[0]),
		strings.Join(setClauses, ", "),
	)

	queryable := r.getQueryable(ctx)
	_, err = queryable.Exec(ctx, query, values...)
	return err
}

// BatchUpsert creates or updates multiple entities using ON CONFLICT
func (r *CockroachDBConnector[T, ID]) BatchUpsert(ctx context.Context, items []T) error {
	if len(items) == 0 {
		return nil
	}

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
				err = commitErr
			}
		}
	}()

	// Build the SET clause for ON CONFLICT DO UPDATE
	var setClauses []string
	numCols := len(r.columns)
	for i := 1; i < numCols; i++ {
		setClauses = append(setClauses, fmt.Sprintf("%s = EXCLUDED.%s",
			quoteIdentifier(r.columns[i]),
			quoteIdentifier(r.columns[i]),
		))
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s",
		quoteIdentifier(r.tableName),
		joinQuotedColumns(r.columns),
		buildPlaceholders(len(r.columns)),
		quoteIdentifier(r.columns[0]),
		strings.Join(setClauses, ", "),
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

// getQueryable returns the queryable (pool or tx) from the context
// If a transaction exists in the context, it returns the transaction
// Otherwise, it returns the pool
func (r *CockroachDBConnector[T, ID]) getQueryable(ctx context.Context) Queryable {
	if tx, ok := getTxFromContext(ctx); ok {
		return tx
	}
	return r.pool
}
