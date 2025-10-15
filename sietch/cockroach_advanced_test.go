package sietch

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/seb7887/gofw/sietch/internal/testutils"
)

func TestCockroachDBQueryBuilderAdvanced(t *testing.T) {
	mockPool := &pgxpool.Pool{}
	conn, err := NewCockroachDBConnector[testutils.Account, int64](
		mockPool,
		"accounts",
		func(a *testutils.Account) int64 { return a.ID },
	)
	if err != nil {
		t.Fatalf("Failed to create connector: %v", err)
	}

	t.Run("OpIn operator", func(t *testing.T) {
		filter := NewFilter().
			Where("id", OpIn, []int64{1, 2, 3}).
			Build()

		query, args, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT "id", "balance" FROM "accounts" WHERE "id" IN ($1, $2, $3)`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}

		if len(args) != 3 {
			t.Errorf("Expected 3 args, got %d", len(args))
		}
	})

	t.Run("OpNotIn operator", func(t *testing.T) {
		filter := NewFilter().
			Where("balance", OpNotIn, []int{100, 200}).
			Build()

		query, args, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT "id", "balance" FROM "accounts" WHERE "balance" NOT IN ($1, $2)`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}

		if len(args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(args))
		}
	})

	t.Run("OpLike operator", func(t *testing.T) {
		filter := NewFilter().
			Where("id", OpLike, "%test%").
			Build()

		query, args, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT "id", "balance" FROM "accounts" WHERE "id" LIKE $1`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}

		if len(args) != 1 {
			t.Errorf("Expected 1 arg, got %d", len(args))
		}
		if args[0] != "%test%" {
			t.Errorf("Expected arg '%%test%%', got %v", args[0])
		}
	})

	t.Run("OpILike operator", func(t *testing.T) {
		filter := NewFilter().
			Where("balance", OpILike, "%TEST%").
			Build()

		query, args, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT "id", "balance" FROM "accounts" WHERE "balance" ILIKE $1`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}

		if len(args) != 1 {
			t.Errorf("Expected 1 arg, got %d", len(args))
		}
	})

	t.Run("OpIsNull operator", func(t *testing.T) {
		filter := NewFilter().
			Where("balance", OpIsNull, nil).
			Build()

		query, args, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT "id", "balance" FROM "accounts" WHERE "balance" IS NULL`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}

		if len(args) != 0 {
			t.Errorf("Expected 0 args for IS NULL, got %d", len(args))
		}
	})

	t.Run("OpIsNotNull operator", func(t *testing.T) {
		filter := NewFilter().
			Where("id", OpIsNotNull, nil).
			Build()

		query, args, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT "id", "balance" FROM "accounts" WHERE "id" IS NOT NULL`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}

		if len(args) != 0 {
			t.Errorf("Expected 0 args for IS NOT NULL, got %d", len(args))
		}
	})

	t.Run("OpBetween operator", func(t *testing.T) {
		filter := NewFilter().
			Where("balance", OpBetween, []int{100, 500}).
			Build()

		query, args, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT "id", "balance" FROM "accounts" WHERE "balance" BETWEEN $1 AND $2`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}

		if len(args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(args))
		}
	})

	t.Run("Multiple conditions with different operators", func(t *testing.T) {
		filter := NewFilter().
			Where("balance", OpGreaterThan, 100).
			Where("id", OpIn, []int64{1, 2, 3}).
			Where("balance", OpLessThan, 1000).
			Build()

		query, args, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT "id", "balance" FROM "accounts" WHERE "balance" > $1 AND "id" IN ($2, $3, $4) AND "balance" < $5`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}

		if len(args) != 5 {
			t.Errorf("Expected 5 args, got %d", len(args))
		}
	})

	t.Run("Sort with multiple fields", func(t *testing.T) {
		filter := NewFilter().
			OrderBy("balance", SortDesc).
			OrderBy("id", SortAsc).
			Build()

		query, args, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT "id", "balance" FROM "accounts" ORDER BY "balance" DESC, "id" ASC`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}

		if len(args) != 0 {
			t.Errorf("Expected 0 args, got %d", len(args))
		}
	})

	t.Run("Limit and Offset", func(t *testing.T) {
		filter := NewFilter().
			Limit(10).
			Offset(20).
			Build()

		query, _, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT "id", "balance" FROM "accounts" LIMIT 10 OFFSET 20`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}
	})

	t.Run("Distinct", func(t *testing.T) {
		filter := NewFilter().
			Distinct().
			Build()

		query, _, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT DISTINCT "id", "balance" FROM "accounts"`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}
	})

	t.Run("Complete query with all features", func(t *testing.T) {
		filter := NewFilter().
			Distinct().
			Where("balance", OpGreaterThan, 100).
			Where("id", OpNotIn, []int64{5, 6}).
			OrderBy("balance", SortDesc).
			Limit(5).
			Offset(10).
			Build()

		query, args, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT DISTINCT "id", "balance" FROM "accounts" WHERE "balance" > $1 AND "id" NOT IN ($2, $3) ORDER BY "balance" DESC LIMIT 5 OFFSET 10`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}

		if len(args) != 3 {
			t.Errorf("Expected 3 args, got %d", len(args))
		}
	})
}

func TestCockroachDBQueryBuilderErrors(t *testing.T) {
	mockPool := &pgxpool.Pool{}
	conn, err := NewCockroachDBConnector[testutils.Account, int64](
		mockPool,
		"accounts",
		func(a *testutils.Account) int64 { return a.ID },
	)
	if err != nil {
		t.Fatalf("Failed to create connector: %v", err)
	}

	t.Run("Invalid field name", func(t *testing.T) {
		filter := NewFilter().
			Where("invalid_field", OpEqual, 100).
			Build()

		_, _, err := conn.queryBuilder(filter)
		if err == nil {
			t.Error("Expected error for invalid field")
		}
	})

	t.Run("Invalid sort field", func(t *testing.T) {
		filter := NewFilter().
			OrderBy("invalid_field", SortAsc).
			Build()

		_, _, err := conn.queryBuilder(filter)
		if err == nil {
			t.Error("Expected error for invalid sort field")
		}
	})

	t.Run("OpIn with non-slice value", func(t *testing.T) {
		filter := NewFilter().
			Where("id", OpIn, 123).
			Build()

		_, _, err := conn.queryBuilder(filter)
		if err == nil {
			t.Error("Expected error for OpIn with non-slice value")
		}
	})

	t.Run("OpNotIn with non-slice value", func(t *testing.T) {
		filter := NewFilter().
			Where("id", OpNotIn, "invalid").
			Build()

		_, _, err := conn.queryBuilder(filter)
		if err == nil {
			t.Error("Expected error for OpNotIn with non-slice value")
		}
	})

	t.Run("OpBetween with non-slice value", func(t *testing.T) {
		filter := NewFilter().
			Where("balance", OpBetween, 100).
			Build()

		_, _, err := conn.queryBuilder(filter)
		if err == nil {
			t.Error("Expected error for OpBetween with non-slice value")
		}
	})

	t.Run("OpBetween with wrong slice length", func(t *testing.T) {
		filter := NewFilter().
			Where("balance", OpBetween, []int{100}).
			Build()

		_, _, err := conn.queryBuilder(filter)
		if err == nil {
			t.Error("Expected error for OpBetween with wrong slice length")
		}
	})

	t.Run("OpIn with empty slice", func(t *testing.T) {
		filter := NewFilter().
			Where("id", OpIn, []int64{}).
			Build()

		_, _, err := conn.queryBuilder(filter)
		if err == nil {
			t.Error("Expected error for OpIn with empty slice")
		}
	})
}

func TestCockroachDBValidateFilterField(t *testing.T) {
	mockPool := &pgxpool.Pool{}
	conn, err := NewCockroachDBConnector[testutils.Account, int64](
		mockPool,
		"accounts",
		func(a *testutils.Account) int64 { return a.ID },
	)
	if err != nil {
		t.Fatalf("Failed to create connector: %v", err)
	}

	t.Run("Valid field names", func(t *testing.T) {
		validFields := []string{"id", "balance"}
		for _, field := range validFields {
			err := conn.validateFilterField(field)
			if err != nil {
				t.Errorf("Expected no error for valid field '%s', got %v", field, err)
			}
		}
	})

	t.Run("Invalid field names", func(t *testing.T) {
		invalidFields := []string{"unknown", "invalid_field", "email"}
		for _, field := range invalidFields {
			err := conn.validateFilterField(field)
			if err == nil {
				t.Errorf("Expected error for invalid field '%s'", field)
			}
		}
	})
}

func TestBuildOrderByClause(t *testing.T) {
	mockPool := &pgxpool.Pool{}
	conn, err := NewCockroachDBConnector[testutils.Account, int64](
		mockPool,
		"accounts",
		func(a *testutils.Account) int64 { return a.ID },
	)
	if err != nil {
		t.Fatalf("Failed to create connector: %v", err)
	}

	t.Run("Single sort field ASC", func(t *testing.T) {
		filter := NewFilter().
			OrderBy("balance", SortAsc).
			Build()

		query, _, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT "id", "balance" FROM "accounts" ORDER BY "balance" ASC`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}
	})

	t.Run("Single sort field DESC", func(t *testing.T) {
		filter := NewFilter().
			OrderBy("id", SortDesc).
			Build()

		query, _, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT "id", "balance" FROM "accounts" ORDER BY "id" DESC`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}
	})

	t.Run("Multiple sort fields", func(t *testing.T) {
		filter := NewFilter().
			OrderBy("balance", SortDesc).
			OrderBy("id", SortAsc).
			Build()

		query, _, err := conn.queryBuilder(filter)
		if err != nil {
			t.Fatalf("queryBuilder failed: %v", err)
		}

		expectedQuery := `SELECT "id", "balance" FROM "accounts" ORDER BY "balance" DESC, "id" ASC`
		if query != expectedQuery {
			t.Errorf("Expected: %s\nGot: %s", expectedQuery, query)
		}
	})
}
