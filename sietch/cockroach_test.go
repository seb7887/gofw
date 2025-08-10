package sietch

import (
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/seb7887/gofw/sietch/internal/testutils"
	"testing"
)

func createTestConnector(t *testing.T) *CockroachDBConnector[testutils.Account, int64] {
	mockPool := &pgxpool.Pool{} // This is just for the test, won't be used for actual DB operations
	conn, err := NewCockroachDBConnector[testutils.Account, int64](
		mockPool,
		"test",
		func(account *testutils.Account) int64 {
			return account.ID
		})
	
	if err != nil {
		t.Fatalf("Failed to create test connector: %s", err)
	}
	
	return conn
}

func TestNewCockroachDBConnector(t *testing.T) {
	// Test with nil pool should fail
	_, err := NewCockroachDBConnector[testutils.Account, int64](
		nil,
		"test",
		func(t *testutils.Account) int64 {
			return t.ID
		})

	if err == nil {
		t.Error("NewCockroachDBConnector should fail with nil pool")
	}

	// Test with nil getID function should fail
	mockPool := &pgxpool.Pool{} // This is just for the test, won't be used
	_, err = NewCockroachDBConnector[testutils.Account, int64](
		mockPool,
		"test",
		nil)

	if err == nil {
		t.Error("NewCockroachDBConnector should fail with nil getID function")
	}

	// Test with invalid table name should fail
	_, err = NewCockroachDBConnector[testutils.Account, int64](
		mockPool,
		"test-invalid",
		func(t *testutils.Account) int64 {
			return t.ID
		})

	if err == nil {
		t.Error("NewCockroachDBConnector should fail with invalid table name")
	}
}

func TestCockroachDBConnector_getValues(t *testing.T) {
	conn := createTestConnector(t)

	item := &testutils.Account{
		ID:      1,
		Balance: 100,
	}
	values, err := conn.getValues(item)
	if err != nil {
		t.Errorf("getValues returned error: %s", err)
	}

	if len(values) != 2 {
		t.Errorf("getValues returned %d values, expected 2", len(values))
	}
}

func TestCockroachDBConnector_getScanDestinations(t *testing.T) {
	conn := createTestConnector(t)

	item := &testutils.Account{
		ID:      1,
		Balance: 100,
	}

	d, err := conn.getScanDestinations(item)
	if err != nil {
		t.Errorf("getScanDestinations returned error: %s", err)
	}

	if len(d) != 2 {
		t.Errorf("getScanDestinations returned %d destinations, expected 2", len(d))
	}
}

func TestCockroachDBConnector_builQuery(t *testing.T) {
	conn := createTestConnector(t)

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		quoteIdentifier(conn.tableName),
		joinQuotedColumns(conn.columns),
		buildPlaceholders(len(conn.columns)),
	)

	expected := `INSERT INTO "test" ("id", "balance") VALUES ($1, $2)`
	if query != expected {
		t.Errorf("expected: %s, got: %s", expected, query)
	}
}

func TestCockroachDBConnector_queryBuilder(t *testing.T) {
	conn := createTestConnector(t)

	filter := &Filter{
		Conditions: []Condition{
			{
				Field:    "email",
				Operator: "=",
				Value:    "test@test.com",
			},
		},
	}

	query, _ := conn.queryBuilder(filter)
	expected := `SELECT "id", "balance" FROM "test" WHERE "email" = $1`
	if query != expected {
		t.Errorf("expected: %s, got: %s", expected, query)
	}
}
