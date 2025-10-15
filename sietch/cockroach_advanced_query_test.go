package sietch

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/seb7887/gofw/sietch/internal/testutils"
)

func TestCockroachDBQueryBuilder_OrConditions(t *testing.T) {
	mockPool := &pgxpool.Pool{}
	conn, err := NewCockroachDBConnector[testutils.Account, int64](
		mockPool,
		"accounts",
		func(a *testutils.Account) int64 { return a.ID },
	)
	if err != nil {
		t.Fatalf("Failed to create connector: %v", err)
	}

	t.Run("Simple OR with two conditions", func(t *testing.T) {
		filter := NewFilter().
			Or(
				Condition{Field: "balance", Operator: OpEqual, Value: 100},
				Condition{Field: "balance", Operator: OpEqual, Value: 200},
			).
			Build()

		query, args, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT "id", "balance" FROM "accounts" WHERE ("balance" = $1 OR "balance" = $2)`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}

		if len(args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(args))
		}
	})

	t.Run("AND + OR combination", func(t *testing.T) {
		// WHERE balance > 100 AND (id = 1 OR id = 2)
		filter := NewFilter().
			Where("balance", OpGreaterThan, 100).
			Or(
				Condition{Field: "id", Operator: OpEqual, Value: int64(1)},
				Condition{Field: "id", Operator: OpEqual, Value: int64(2)},
			).
			Build()

		query, args, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT "id", "balance" FROM "accounts" WHERE "balance" > $1 AND ("id" = $2 OR "id" = $3)`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}

		if len(args) != 3 {
			t.Errorf("Expected 3 args, got %d", len(args))
		}
	})

	t.Run("Multiple OR groups", func(t *testing.T) {
		// WHERE (balance = 100 OR balance = 200) AND (id = 1 OR id = 2)
		filter := NewFilter().
			Or(
				Condition{Field: "balance", Operator: OpEqual, Value: 100},
				Condition{Field: "balance", Operator: OpEqual, Value: 200},
			).
			Or(
				Condition{Field: "id", Operator: OpEqual, Value: int64(1)},
				Condition{Field: "id", Operator: OpEqual, Value: int64(2)},
			).
			Build()

		query, args, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT "id", "balance" FROM "accounts" WHERE ("balance" = $1 OR "balance" = $2) AND ("id" = $3 OR "id" = $4)`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}

		if len(args) != 4 {
			t.Errorf("Expected 4 args, got %d", len(args))
		}
	})
}

func TestCockroachDBQueryBuilder_NotConditions(t *testing.T) {
	mockPool := &pgxpool.Pool{}
	conn, err := NewCockroachDBConnector[testutils.Account, int64](
		mockPool,
		"accounts",
		func(a *testutils.Account) int64 { return a.ID },
	)
	if err != nil {
		t.Fatalf("Failed to create connector: %v", err)
	}

	t.Run("NOT with simple condition", func(t *testing.T) {
		filter := NewFilter().
			Not(Condition{Field: "balance", Operator: OpEqual, Value: 100}).
			Build()

		query, args, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT "id", "balance" FROM "accounts" WHERE NOT ("balance" = $1)`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}

		if len(args) != 1 {
			t.Errorf("Expected 1 arg, got %d", len(args))
		}
	})

	t.Run("NOT with OR condition", func(t *testing.T) {
		// WHERE NOT (balance = 100 OR balance = 200)
		filter := NewFilter().
			Not(Condition{
				LogicalOp: LogicalOR,
				Conditions: []Condition{
					{Field: "balance", Operator: OpEqual, Value: 100},
					{Field: "balance", Operator: OpEqual, Value: 200},
				},
			}).
			Build()

		query, args, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT "id", "balance" FROM "accounts" WHERE NOT (("balance" = $1 OR "balance" = $2))`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}

		if len(args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(args))
		}
	})

	t.Run("NOT with AND condition", func(t *testing.T) {
		// WHERE NOT (balance > 100 AND id < 10)
		filter := NewFilter().
			Not(Condition{
				LogicalOp: LogicalAND,
				Conditions: []Condition{
					{Field: "balance", Operator: OpGreaterThan, Value: 100},
					{Field: "id", Operator: OpLessThan, Value: int64(10)},
				},
			}).
			Build()

		query, args, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT "id", "balance" FROM "accounts" WHERE NOT (("balance" > $1 AND "id" < $2))`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}

		if len(args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(args))
		}
	})
}

func TestCockroachDBQueryBuilder_ComplexNested(t *testing.T) {
	mockPool := &pgxpool.Pool{}
	conn, err := NewCockroachDBConnector[testutils.Account, int64](
		mockPool,
		"accounts",
		func(a *testutils.Account) int64 { return a.ID },
	)
	if err != nil {
		t.Fatalf("Failed to create connector: %v", err)
	}

	t.Run("Complex nested conditions", func(t *testing.T) {
		// WHERE (balance > 100 OR balance < 50) AND NOT (id = 5)
		filter := NewFilter().
			Or(
				Condition{Field: "balance", Operator: OpGreaterThan, Value: 100},
				Condition{Field: "balance", Operator: OpLessThan, Value: 50},
			).
			Not(Condition{Field: "id", Operator: OpEqual, Value: int64(5)}).
			Build()

		query, args, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT "id", "balance" FROM "accounts" WHERE ("balance" > $1 OR "balance" < $2) AND NOT ("id" = $3)`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}

		if len(args) != 3 {
			t.Errorf("Expected 3 args, got %d", len(args))
		}
	})

	t.Run("AND nested inside OR", func(t *testing.T) {
		// WHERE (balance > 100 AND balance < 200) OR (id > 10 AND id < 20)
		filter := NewFilter().
			Or(
				Condition{
					LogicalOp: LogicalAND,
					Conditions: []Condition{
						{Field: "balance", Operator: OpGreaterThan, Value: 100},
						{Field: "balance", Operator: OpLessThan, Value: 200},
					},
				},
				Condition{
					LogicalOp: LogicalAND,
					Conditions: []Condition{
						{Field: "id", Operator: OpGreaterThan, Value: int64(10)},
						{Field: "id", Operator: OpLessThan, Value: int64(20)},
					},
				},
			).
			Build()

		query, args, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT "id", "balance" FROM "accounts" WHERE (("balance" > $1 AND "balance" < $2) OR ("id" > $3 AND "id" < $4))`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}

		if len(args) != 4 {
			t.Errorf("Expected 4 args, got %d", len(args))
		}
	})

	t.Run("Complex with sorting and pagination", func(t *testing.T) {
		// WHERE balance > 100 AND (id = 1 OR id = 2) ORDER BY balance DESC LIMIT 10 OFFSET 5
		filter := NewFilter().
			Where("balance", OpGreaterThan, 100).
			Or(
				Condition{Field: "id", Operator: OpEqual, Value: int64(1)},
				Condition{Field: "id", Operator: OpEqual, Value: int64(2)},
			).
			OrderBy("balance", SortDesc).
			Limit(10).
			Offset(5).
			Build()

		query, args, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT "id", "balance" FROM "accounts" WHERE "balance" > $1 AND ("id" = $2 OR "id" = $3) ORDER BY "balance" DESC LIMIT 10 OFFSET 5`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}

		if len(args) != 3 {
			t.Errorf("Expected 3 args, got %d", len(args))
		}
	})
}

func TestCockroachDBQueryBuilder_AdvancedOperatorsWithLogical(t *testing.T) {
	mockPool := &pgxpool.Pool{}
	conn, err := NewCockroachDBConnector[testutils.Account, int64](
		mockPool,
		"accounts",
		func(a *testutils.Account) int64 { return a.ID },
	)
	if err != nil {
		t.Fatalf("Failed to create connector: %v", err)
	}

	t.Run("OR with IN operator", func(t *testing.T) {
		// WHERE (balance IN (100, 200)) OR (id > 10)
		filter := NewFilter().
			Or(
				Condition{Field: "balance", Operator: OpIn, Value: []int{100, 200}},
				Condition{Field: "id", Operator: OpGreaterThan, Value: int64(10)},
			).
			Build()

		query, args, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT "id", "balance" FROM "accounts" WHERE ("balance" IN ($1, $2) OR "id" > $3)`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}

		if len(args) != 3 {
			t.Errorf("Expected 3 args, got %d", len(args))
		}
	})

	t.Run("NOT with BETWEEN", func(t *testing.T) {
		// WHERE NOT (balance BETWEEN 100 AND 200)
		filter := NewFilter().
			Not(Condition{Field: "balance", Operator: OpBetween, Value: []int{100, 200}}).
			Build()

		query, args, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT "id", "balance" FROM "accounts" WHERE NOT ("balance" BETWEEN $1 AND $2)`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}

		if len(args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(args))
		}
	})
}
