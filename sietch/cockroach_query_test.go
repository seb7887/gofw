package sietch

import (
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/seb7887/gofw/sietch/internal/testutils"
	"strings"
	"testing"
)

// Helper to create connector for query testing
func createQueryTestConnector(t *testing.T, tableName string) *CockroachDBConnector[testutils.Account, int64] {
	mockPool := &pgxpool.Pool{} // Won't be used for actual queries in these tests
	conn, err := NewCockroachDBConnector[testutils.Account, int64](
		mockPool,
		tableName,
		func(account *testutils.Account) int64 {
			return account.ID
		})
	
	if err != nil {
		t.Fatalf("Failed to create test connector: %s", err)
	}
	
	return conn
}

// Test Create query formation
func TestCockroachDBConnector_CreateQueryFormat(t *testing.T) {
	conn := createQueryTestConnector(t, "accounts")
	
	// Test the query building by accessing internal methods
	values, err := conn.getValues(&testutils.Account{ID: 1, Balance: 100})
	if err != nil {
		t.Fatalf("getValues failed: %v", err)
	}
	
	// Build the query manually to test format
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		quoteIdentifier(conn.tableName),
		joinQuotedColumns(conn.columns),
		buildPlaceholders(len(conn.columns)),
	)
	
	expectedQuery := `INSERT INTO "accounts" ("id", "balance") VALUES ($1, $2)`
	if query != expectedQuery {
		t.Errorf("Create query format incorrect:\nExpected: %s\nGot: %s", expectedQuery, query)
	}
	
	// Verify values are correct
	expectedValues := []any{int64(1), 100}
	if len(values) != len(expectedValues) {
		t.Fatalf("Expected %d values, got %d", len(expectedValues), len(values))
	}
	
	for i, expected := range expectedValues {
		if values[i] != expected {
			t.Errorf("Expected value[%d]: %v, got: %v", i, expected, values[i])
		}
	}
}

// Test Get query formation
func TestCockroachDBConnector_GetQueryFormat(t *testing.T) {
	conn := createQueryTestConnector(t, "users")
	
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1",
		joinQuotedColumns(conn.columns),
		quoteIdentifier(conn.tableName),
		quoteIdentifier(conn.columns[0]),
	)
	
	expectedQuery := `SELECT "id", "balance" FROM "users" WHERE "id" = $1`
	if query != expectedQuery {
		t.Errorf("Get query format incorrect:\nExpected: %s\nGot: %s", expectedQuery, query)
	}
}

// Test Update query formation
func TestCockroachDBConnector_UpdateQueryFormat(t *testing.T) {
	conn := createQueryTestConnector(t, "profiles")
	
	item := &testutils.Account{ID: 5, Balance: 250}
	values, err := conn.getValues(item)
	if err != nil {
		t.Fatalf("getValues failed: %v", err)
	}
	
	// Build update query
	var setClause []string
	numCols := len(conn.columns)
	for i := 1; i < numCols; i++ {
		setClause = append(setClause, fmt.Sprintf("%s = $%d", quoteIdentifier(conn.columns[i]), i))
	}
	
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = $%d",
		quoteIdentifier(conn.tableName),
		strings.Join(setClause, ", "),
		quoteIdentifier(conn.columns[0]),
		numCols,
	)
	
	expectedQuery := `UPDATE "profiles" SET "balance" = $1 WHERE "id" = $2`
	if query != expectedQuery {
		t.Errorf("Update query format incorrect:\nExpected: %s\nGot: %s", expectedQuery, query)
	}
	
	// Verify values and ID extraction
	id := conn.getID(item)
	updateArgs := append(values[1:], id)
	expectedArgs := []any{250, int64(5)}
	
	if len(updateArgs) != len(expectedArgs) {
		t.Fatalf("Expected %d args, got %d", len(expectedArgs), len(updateArgs))
	}
	
	for i, expected := range expectedArgs {
		if updateArgs[i] != expected {
			t.Errorf("Expected arg[%d]: %v, got: %v", i, expected, updateArgs[i])
		}
	}
}

// Test Delete query formation
func TestCockroachDBConnector_DeleteQueryFormat(t *testing.T) {
	conn := createQueryTestConnector(t, "orders")
	
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = $1",
		quoteIdentifier(conn.tableName),
		quoteIdentifier(conn.columns[0]),
	)
	
	expectedQuery := `DELETE FROM "orders" WHERE "id" = $1`
	if query != expectedQuery {
		t.Errorf("Delete query format incorrect:\nExpected: %s\nGot: %s", expectedQuery, query)
	}
}

// Test Query with conditions
func TestCockroachDBConnector_QueryWithConditionsFormat(t *testing.T) {
	conn := createQueryTestConnector(t, "customers")
	
	filter := &Filter{
		Conditions: []Condition{
			{Field: "balance", Operator: ">", Value: 100},
			{Field: "id", Operator: "<=", Value: 50},
		},
	}
	
	query, args := conn.queryBuilder(filter)
	
	expectedQuery := `SELECT "id", "balance" FROM "customers" WHERE "balance" > $1 AND "id" <= $2`
	if query != expectedQuery {
		t.Errorf("Query with conditions format incorrect:\nExpected: %s\nGot: %s", expectedQuery, query)
	}
	
	expectedArgs := []any{100, 50}
	if len(args) != len(expectedArgs) {
		t.Fatalf("Expected %d args, got %d", len(expectedArgs), len(args))
	}
	
	for i, expected := range expectedArgs {
		if args[i] != expected {
			t.Errorf("Expected arg[%d]: %v, got: %v", i, expected, args[i])
		}
	}
}

// Test Query with empty filter
func TestCockroachDBConnector_QueryEmptyFilterFormat(t *testing.T) {
	conn := createQueryTestConnector(t, "products")
	
	filter := &Filter{Conditions: []Condition{}}
	
	query, args := conn.queryBuilder(filter)
	
	expectedQuery := `SELECT "id", "balance" FROM "products"`
	if query != expectedQuery {
		t.Errorf("Query with empty filter format incorrect:\nExpected: %s\nGot: %s", expectedQuery, query)
	}
	
	if len(args) != 0 {
		t.Errorf("Expected 0 args for empty filter, got %d", len(args))
	}
}

// Test Query with nil filter
func TestCockroachDBConnector_QueryNilFilterFormat(t *testing.T) {
	conn := createQueryTestConnector(t, "inventory")
	
	query, args := conn.queryBuilder(nil)
	
	expectedQuery := `SELECT "id", "balance" FROM "inventory"`
	if query != expectedQuery {
		t.Errorf("Query with nil filter format incorrect:\nExpected: %s\nGot: %s", expectedQuery, query)
	}
	
	if len(args) != 0 {
		t.Errorf("Expected 0 args for nil filter, got %d", len(args))
	}
}

// Test single condition query
func TestCockroachDBConnector_QuerySingleConditionFormat(t *testing.T) {
	conn := createQueryTestConnector(t, "transactions")
	
	filter := &Filter{
		Conditions: []Condition{
			{Field: "balance", Operator: "=", Value: 500},
		},
	}
	
	query, args := conn.queryBuilder(filter)
	
	expectedQuery := `SELECT "id", "balance" FROM "transactions" WHERE "balance" = $1`
	if query != expectedQuery {
		t.Errorf("Query with single condition format incorrect:\nExpected: %s\nGot: %s", expectedQuery, query)
	}
	
	expectedArgs := []any{500}
	if len(args) != len(expectedArgs) {
		t.Fatalf("Expected %d args, got %d", len(expectedArgs), len(args))
	}
	
	if args[0] != expectedArgs[0] {
		t.Errorf("Expected arg: %v, got: %v", expectedArgs[0], args[0])
	}
}

// Test multiple conditions with different operators
func TestCockroachDBConnector_QueryMultipleOperatorsFormat(t *testing.T) {
	conn := createQueryTestConnector(t, "accounts")
	
	filter := &Filter{
		Conditions: []Condition{
			{Field: "balance", Operator: ">=", Value: 1000},
			{Field: "id", Operator: "!=", Value: 999},
			{Field: "balance", Operator: "<", Value: 5000},
		},
	}
	
	query, args := conn.queryBuilder(filter)
	
	expectedQuery := `SELECT "id", "balance" FROM "accounts" WHERE "balance" >= $1 AND "id" != $2 AND "balance" < $3`
	if query != expectedQuery {
		t.Errorf("Query with multiple operators format incorrect:\nExpected: %s\nGot: %s", expectedQuery, query)
	}
	
	expectedArgs := []any{1000, 999, 5000}
	if len(args) != len(expectedArgs) {
		t.Fatalf("Expected %d args, got %d", len(expectedArgs), len(args))
	}
	
	for i, expected := range expectedArgs {
		if args[i] != expected {
			t.Errorf("Expected arg[%d]: %v, got: %v", i, expected, args[i])
		}
	}
}

// Test SQL injection protection in table names
func TestCockroachDBConnector_TableNameValidation(t *testing.T) {
	mockPool := &pgxpool.Pool{}
	
	tests := []struct {
		tableName   string
		shouldFail  bool
		description string
	}{
		{"valid_table", false, "valid table name"},
		{"ValidTable123", false, "valid table name with numbers"},
		{"users_accounts", false, "valid table name with underscore"},
		{"_internal", false, "valid table name starting with underscore"},
		{"table123_test", false, "valid table name with numbers and underscore"},
		{"table-name", true, "table name with hyphen should fail"},
		{"table name", true, "table name with space should fail"},
		{"table;DROP", true, "table name with semicolon should fail"},
		{"table'test", true, "table name with quote should fail"},
		{"table\"test", true, "table name with double quote should fail"},
		{"table(test)", true, "table name with parentheses should fail"},
		{"table*test", true, "table name with asterisk should fail"},
		{"", true, "empty table name should fail"},
	}
	
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			_, err := NewCockroachDBConnector[testutils.Account, int64](
				mockPool,
				tt.tableName,
				func(account *testutils.Account) int64 {
					return account.ID
				})
			
			if tt.shouldFail && err == nil {
				t.Errorf("Expected error for table name '%s', but got none", tt.tableName)
			} else if !tt.shouldFail && err != nil {
				t.Errorf("Expected no error for table name '%s', but got: %v", tt.tableName, err)
			}
		})
	}
}

// Test that all identifiers are properly quoted
func TestCockroachDBConnector_IdentifierQuoting(t *testing.T) {
	conn := createQueryTestConnector(t, "test_table")
	
	tests := []struct {
		name     string
		buildQuery func() string
		description string
	}{
		{
			name: "CREATE",
			buildQuery: func() string {
				return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
					quoteIdentifier(conn.tableName),
					joinQuotedColumns(conn.columns),
					buildPlaceholders(len(conn.columns)))
			},
			description: "Create query should have quoted identifiers",
		},
		{
			name: "GET",
			buildQuery: func() string {
				return fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1",
					joinQuotedColumns(conn.columns),
					quoteIdentifier(conn.tableName),
					quoteIdentifier(conn.columns[0]))
			},
			description: "Get query should have quoted identifiers",
		},
		{
			name: "UPDATE",
			buildQuery: func() string {
				var setClause []string
				for i := 1; i < len(conn.columns); i++ {
					setClause = append(setClause, fmt.Sprintf("%s = $%d", quoteIdentifier(conn.columns[i]), i))
				}
				return fmt.Sprintf("UPDATE %s SET %s WHERE %s = $%d",
					quoteIdentifier(conn.tableName),
					strings.Join(setClause, ", "),
					quoteIdentifier(conn.columns[0]),
					len(conn.columns)+1)
			},
			description: "Update query should have quoted identifiers",
		},
		{
			name: "DELETE",
			buildQuery: func() string {
				return fmt.Sprintf("DELETE FROM %s WHERE %s = $1",
					quoteIdentifier(conn.tableName),
					quoteIdentifier(conn.columns[0]))
			},
			description: "Delete query should have quoted identifiers",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := tt.buildQuery()
			
			// Check that table name is quoted
			if !strings.Contains(query, `"test_table"`) {
				t.Errorf("%s: Table name not properly quoted in query: %s", tt.description, query)
			}
			
			// Check that column names are quoted
			if !strings.Contains(query, `"id"`) {
				t.Errorf("%s: id column not properly quoted in query: %s", tt.description, query)
			}
			
			// DELETE queries only use id column, others should have balance
			if tt.name != "DELETE" && !strings.Contains(query, `"balance"`) {
				t.Errorf("%s: balance column not properly quoted in query: %s", tt.description, query)
			}
			
			// Ensure no unquoted identifiers (basic check)
			if strings.Contains(query, " test_table ") || strings.Contains(query, " id ") || strings.Contains(query, " balance ") {
				t.Errorf("%s: Found unquoted identifiers in query: %s", tt.description, query)
			}
		})
	}
}

// Test query builder with filter conditions quoting
func TestCockroachDBConnector_FilterFieldQuoting(t *testing.T) {
	conn := createQueryTestConnector(t, "test_filter")
	
	filter := &Filter{
		Conditions: []Condition{
			{Field: "balance", Operator: "=", Value: 100},
			{Field: "id", Operator: ">", Value: 0},
		},
	}
	
	query, _ := conn.queryBuilder(filter)
	
	// Check that field names in conditions are quoted
	if !strings.Contains(query, `"balance" =`) {
		t.Errorf("Filter field 'balance' not properly quoted in query: %s", query)
	}
	
	if !strings.Contains(query, `"id" >`) {
		t.Errorf("Filter field 'id' not properly quoted in query: %s", query)
	}
	
	// Check table name is quoted
	if !strings.Contains(query, `"test_filter"`) {
		t.Errorf("Table name not properly quoted in query: %s", query)
	}
	
	// Check column names in SELECT are quoted
	if !strings.Contains(query, `SELECT "id", "balance"`) {
		t.Errorf("SELECT columns not properly quoted in query: %s", query)
	}
}

// Test edge cases for placeholder generation
func TestCockroachDBConnector_PlaceholderGeneration(t *testing.T) {
	tests := []struct {
		count    int
		expected string
	}{
		{1, "$1"},
		{2, "$1, $2"},
		{3, "$1, $2, $3"},
		{5, "$1, $2, $3, $4, $5"},
	}
	
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d_placeholders", tt.count), func(t *testing.T) {
			result := buildPlaceholders(tt.count)
			if result != tt.expected {
				t.Errorf("Expected placeholders: %s, got: %s", tt.expected, result)
			}
		})
	}
}

// Test column extraction from struct tags
func TestCockroachDBConnector_ColumnExtraction(t *testing.T) {
	columns, err := getColumns[testutils.Account]()
	if err != nil {
		t.Fatalf("getColumns failed: %v", err)
	}
	
	expectedColumns := []string{"id", "balance"}
	if len(columns) != len(expectedColumns) {
		t.Fatalf("Expected %d columns, got %d", len(expectedColumns), len(columns))
	}
	
	for i, expected := range expectedColumns {
		if columns[i] != expected {
			t.Errorf("Expected column[%d]: %s, got: %s", i, expected, columns[i])
		}
	}
}

// Test column joining functions
func TestCockroachDBConnector_ColumnJoining(t *testing.T) {
	columns := []string{"id", "name", "email"}
	
	// Test regular joining
	regular := joinColumns(columns)
	expectedRegular := "id, name, email"
	if regular != expectedRegular {
		t.Errorf("joinColumns - Expected: %s, got: %s", expectedRegular, regular)
	}
	
	// Test quoted joining
	quoted := joinQuotedColumns(columns)
	expectedQuoted := `"id", "name", "email"`
	if quoted != expectedQuoted {
		t.Errorf("joinQuotedColumns - Expected: %s, got: %s", expectedQuoted, quoted)
	}
}

// Test quote identifier function
func TestCockroachDBConnector_QuoteIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"table", `"table"`},
		{"column_name", `"column_name"`},
		{"CamelCase", `"CamelCase"`},
		{"test123", `"test123"`},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := quoteIdentifier(tt.input)
			if result != tt.expected {
				t.Errorf("quoteIdentifier(%s) - Expected: %s, got: %s", tt.input, tt.expected, result)
			}
		})
	}
}