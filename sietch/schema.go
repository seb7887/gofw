package sietch

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

// ColumnType represents SQL column data types
type ColumnType string

const (
	ColumnTypeSerial    ColumnType = "SERIAL"
	ColumnTypeBigSerial ColumnType = "BIGSERIAL"
	ColumnTypeInteger   ColumnType = "INTEGER"
	ColumnTypeBigInt    ColumnType = "BIGINT"
	ColumnTypeText      ColumnType = "TEXT"
	ColumnTypeVarchar   ColumnType = "VARCHAR"
	ColumnTypeBoolean   ColumnType = "BOOLEAN"
	ColumnTypeTimestamp ColumnType = "TIMESTAMP"
	ColumnTypeDate      ColumnType = "DATE"
	ColumnTypeJSON      ColumnType = "JSONB"
	ColumnTypeFloat     ColumnType = "FLOAT8"
	ColumnTypeNumeric   ColumnType = "NUMERIC"
)

// IndexType represents different types of database indexes
type IndexType string

const (
	IndexTypeBTree IndexType = "BTREE"
	IndexTypeHash  IndexType = "HASH"
	IndexTypeGin   IndexType = "GIN"
	IndexTypeGist  IndexType = "GIST"
)

// ColumnDef defines a table column
type ColumnDef struct {
	Name         string
	Type         ColumnType
	PrimaryKey   bool
	NotNull      bool
	Unique       bool
	DefaultValue string
	Check        string
}

// IndexDef defines a table index
type IndexDef struct {
	Name    string
	Type    IndexType
	Columns []string
	Unique  bool
	Where   string // Partial index condition
}

// TableDef defines a complete table schema
type TableDef struct {
	Name    string
	Columns []ColumnDef
	Indexes []IndexDef
}

// SchemaHelper provides utilities for schema management (primarily for testing)
type SchemaHelper struct {
	connector *CockroachDBConnector[any, any]
}

// NewSchemaHelper creates a new schema helper
// Note: This is primarily for testing and development
func NewSchemaHelper[T any, ID comparable](connector *CockroachDBConnector[T, ID]) *SchemaHelper {
	// Type assertion to work with any type
	anyConnector := (*CockroachDBConnector[any, any])(nil)
	return &SchemaHelper{
		connector: anyConnector,
	}
}

// InferTableDef infers table definition from a struct type
func InferTableDef[T any](tableName string) (*TableDef, error) {
	var zero T
	typ := reflect.TypeOf(zero)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("type must be a struct")
	}

	tableDef := &TableDef{
		Name:    tableName,
		Columns: make([]ColumnDef, 0),
		Indexes: make([]IndexDef, 0),
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag == "" || dbTag == "-" {
			continue
		}

		colDef := ColumnDef{
			Name:       dbTag,
			Type:       inferColumnType(field.Type),
			PrimaryKey: i == 0, // First field is assumed to be primary key
			NotNull:    true,
		}

		// Check for additional tags
		if field.Tag.Get("unique") == "true" {
			colDef.Unique = true
		}
		if field.Tag.Get("nullable") == "true" {
			colDef.NotNull = false
		}
		if defaultVal := field.Tag.Get("default"); defaultVal != "" {
			colDef.DefaultValue = defaultVal
		}

		tableDef.Columns = append(tableDef.Columns, colDef)
	}

	return tableDef, nil
}

// inferColumnType maps Go types to SQL column types
func inferColumnType(t reflect.Type) ColumnType {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Int, reflect.Int32:
		return ColumnTypeInteger
	case reflect.Int64:
		return ColumnTypeBigInt
	case reflect.String:
		return ColumnTypeText
	case reflect.Bool:
		return ColumnTypeBoolean
	case reflect.Float32, reflect.Float64:
		return ColumnTypeFloat
	default:
		if t.String() == "time.Time" {
			return ColumnTypeTimestamp
		}
		return ColumnTypeText
	}
}

// GenerateCreateTableSQL generates CREATE TABLE SQL from table definition
func GenerateCreateTableSQL(def *TableDef) string {
	var parts []string

	// Column definitions
	for _, col := range def.Columns {
		colDef := fmt.Sprintf(`"%s" %s`, col.Name, col.Type)

		if col.PrimaryKey {
			colDef += " PRIMARY KEY"
		}
		if col.NotNull && !col.PrimaryKey {
			colDef += " NOT NULL"
		}
		if col.Unique && !col.PrimaryKey {
			colDef += " UNIQUE"
		}
		if col.DefaultValue != "" {
			colDef += " DEFAULT " + col.DefaultValue
		}
		if col.Check != "" {
			colDef += " CHECK (" + col.Check + ")"
		}

		parts = append(parts, colDef)
	}

	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS \"%s\" (\n  %s\n)",
		def.Name,
		strings.Join(parts, ",\n  "),
	)

	return sql
}

// GenerateDropTableSQL generates DROP TABLE SQL
func GenerateDropTableSQL(tableName string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS \"%s\" CASCADE", tableName)
}

// GenerateCreateIndexSQL generates CREATE INDEX SQL from index definition
func GenerateCreateIndexSQL(tableName string, idx *IndexDef) string {
	uniqueClause := ""
	if idx.Unique {
		uniqueClause = "UNIQUE "
	}

	columns := make([]string, len(idx.Columns))
	for i, col := range idx.Columns {
		columns[i] = fmt.Sprintf(`"%s"`, col)
	}

	sql := fmt.Sprintf("CREATE %sINDEX IF NOT EXISTS \"%s\" ON \"%s\" USING %s (%s)",
		uniqueClause,
		idx.Name,
		tableName,
		idx.Type,
		strings.Join(columns, ", "),
	)

	if idx.Where != "" {
		sql += " WHERE " + idx.Where
	}

	return sql
}

// CreateTableFromStruct creates a table based on a struct definition
// This is primarily for testing and development purposes
func CreateTableFromStruct[T any](ctx context.Context, connector *CockroachDBConnector[T, any], tableName string) error {
	tableDef, err := InferTableDef[T](tableName)
	if err != nil {
		return err
	}

	sql := GenerateCreateTableSQL(tableDef)
	_, err = connector.pool.Exec(ctx, sql)
	return err
}

// DropTable drops a table if it exists
func DropTable(ctx context.Context, connector *CockroachDBConnector[any, any], tableName string) error {
	sql := GenerateDropTableSQL(tableName)
	_, err := connector.pool.Exec(ctx, sql)
	return err
}

// CreateIndex creates an index on a table
func CreateIndex(ctx context.Context, connector *CockroachDBConnector[any, any], tableName string, idx *IndexDef) error {
	sql := GenerateCreateIndexSQL(tableName, idx)
	_, err := connector.pool.Exec(ctx, sql)
	return err
}

// TruncateTable removes all rows from a table
func TruncateTable(ctx context.Context, connector *CockroachDBConnector[any, any], tableName string) error {
	sql := fmt.Sprintf("TRUNCATE TABLE \"%s\" CASCADE", tableName)
	_, err := connector.pool.Exec(ctx, sql)
	return err
}
