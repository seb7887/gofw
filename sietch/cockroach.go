package sietch

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
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

// NewCockroachDBConnector CockroackDB implementation of Repository interface
func NewCockroachDBConnector[T any, ID comparable](pool *pgxpool.Pool, tableName string, getID func(*T) ID) (*CockroachDBConnector[T, ID], error) {
	columns, err := getColumns[T]()
	if err != nil {
		return nil, err
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

func buildPlaceholders(n int) string {
	placeholders := make([]string, n)
	for i := 0; i < n; i++ {
		placeholders[i-1] = fmt.Sprintf("$%d", i)
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
		if tag == "" {
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
		if tag == "" {
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
		r.tableName,
		joinColumns(r.columns),
		buildPlaceholders(len(r.columns)),
	)
	_, err = r.pool.Exec(ctx, query, values...)
	return err
}

func (r *CockroachDBConnector[T, ID]) Get(ctx context.Context, id ID) (*T, error) {
	var t T
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1",
		joinColumns(r.columns),
		r.tableName,
		r.columns[0],
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
			_ = tx.Rollback(ctx)
		} else {
			_ = tx.Commit(ctx)
		}
	}()

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		r.tableName,
		joinColumns(r.columns),
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
	whereClause := ""
	var args []any
	for i, condition := range filter.Conditions {
		if i == 0 {
			whereClause = "WHERE "
		} else {
			whereClause += " AND "
		}
		whereClause += fmt.Sprintf("%s %s %d", condition.Field, condition.Operator, i+1)
		args = append(args, condition.Value)
	}

	query := fmt.Sprintf("SELECT %s FROM %s %s",
		joinColumns(r.columns),
		r.tableName,
		whereClause,
	)
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
		setClause = append(setClause, fmt.Sprintf("%s = $%d", r.columns[i], i))
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = $%d",
		r.tableName,
		strings.Join(setClause, ", "),
		r.columns[0],
		numCols+1,
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
			_ = tx.Rollback(ctx)
		} else {
			_ = tx.Commit(ctx)
		}
	}()

	numCols := len(r.columns)
	var setClauses []string
	for i := 1; i < numCols; i++ {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", r.columns[i], i))
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = $%d",
		r.tableName,
		strings.Join(setClauses, ", "),
		r.columns[0],
		numCols+1,
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
		ct, err := r.pool.Exec(ctx, "batch_update_stmt", args...)
		if err != nil {
			return err
		}

		if ct.RowsAffected() == 0 {
			return errors.New(fmt.Sprintf("batch update item %v does not exist", item))
		}
	}

	return nil
}

func (r *CockroachDBConnector[T, ID]) Delete(ctx context.Context, id ID) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = $1",
		r.tableName,
		r.columns[0],
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
			_ = tx.Rollback(ctx)
		} else {
			_ = tx.Commit(ctx)
		}
	}()

	query := fmt.Sprintf("DELETE FROM %s WHERE %s = $1",
		r.tableName,
		r.columns[0],
	)
	_, err = tx.Prepare(ctx, "batch_delete_stmt", query)
	if err != nil {
		return err
	}

	for _, id := range items {
		ct, err := r.pool.Exec(ctx, "batch_delete_stmt", id)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return errors.New(fmt.Sprintf("%v row not deleted", id))
		}
	}

	return nil
}
