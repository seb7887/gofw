package sietch

import (
	"fmt"
	"github.com/seb7887/gofw/sietch/internal/testutils"
	"testing"
)

func TestNewCockroachDBConnector(t *testing.T) {
	conn, err := NewCockroachDBConnector[testutils.Account, int64](
		nil,
		"test",
		func(t *testutils.Account) int64 {
			return t.ID
		})

	if err != nil {
		t.Errorf("NewCockroachDBConnector returned error: %s", err)
	}

	if len(conn.columns) != 2 {
		t.Errorf("NewCockroachDBConnector returned %d columns, expected 2", len(conn.columns))
	}
}

func TestCockroachDBConnector_getValues(t *testing.T) {
	conn, err := NewCockroachDBConnector[testutils.Account, int64](
		nil,
		"test",
		func(t *testutils.Account) int64 {
			return t.ID
		})

	if err != nil {
		t.Errorf("NewCockroachDBConnector returned error: %s", err)
	}

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
	conn, err := NewCockroachDBConnector[testutils.Account, int64](
		nil,
		"test",
		func(t *testutils.Account) int64 {
			return t.ID
		})

	if err != nil {
		t.Errorf("NewCockroachDBConnector returned error: %s", err)
	}

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
	conn, err := NewCockroachDBConnector[testutils.Account, int64](
		nil,
		"test",
		func(t *testutils.Account) int64 {
			return t.ID
		})

	if err != nil {
		t.Errorf("NewCockroachDBConnector returned error: %s", err)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		conn.tableName,
		joinColumns(conn.columns),
		buildPlaceholders(len(conn.columns)),
	)

	expected := "INSERT INTO test (id, balance) VALUES ($1, $2)"
	if query != expected {
		t.Errorf("expected: %s, got: %s", expected, query)
	}
}

func TestCockroachDBConnector_queryBuilder(t *testing.T) {
	conn, err := NewCockroachDBConnector[testutils.Account, int64](
		nil,
		"test",
		func(t *testutils.Account) int64 {
			return t.ID
		})

	if err != nil {
		t.Errorf("NewCockroachDBConnector returned error: %s", err)
	}

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
	expected := "SELECT id, balance FROM test WHERE email = $1"
	if query != expected {
		t.Errorf("expected: %s, got: %s", expected, query)
	}
}
