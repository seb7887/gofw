package sietch

import (
	"context"
	"testing"
)

func TestInMemoryLikeOperator(t *testing.T) {
	ctx := context.Background()

	t.Run("OpLike pattern matching", func(t *testing.T) {
		type TestEntity struct {
			ID    int64  `db:"id"`
			Email string `db:"email"`
		}

		repo := NewInMemoryConnector[TestEntity, int64](
			func(e *TestEntity) int64 { return e.ID },
		)

		entities := []TestEntity{
			{ID: 1, Email: "user@example.com"},
			{ID: 2, Email: "admin@example.com"},
			{ID: 3, Email: "test@other.com"},
			{ID: 4, Email: "user@other.org"},
		}
		repo.BatchCreate(ctx, entities)

		// Test % wildcard at end
		filter := NewFilter().
			Where("email", OpLike, "%@example.com").
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}
	})

	t.Run("OpLike with % at beginning", func(t *testing.T) {
		type TestEntity struct {
			ID   int64  `db:"id"`
			Name string `db:"name"`
		}

		repo := NewInMemoryConnector[TestEntity, int64](
			func(e *TestEntity) int64 { return e.ID },
		)

		entities := []TestEntity{
			{ID: 1, Name: "John Doe"},
			{ID: 2, Name: "Jane Doe"},
			{ID: 3, Name: "Bob Smith"},
		}
		repo.BatchCreate(ctx, entities)

		filter := NewFilter().
			Where("name", OpLike, "%Doe").
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}
	})

	t.Run("OpLike with % at both ends", func(t *testing.T) {
		type TestEntity struct {
			ID   int64  `db:"id"`
			Text string `db:"text"`
		}

		repo := NewInMemoryConnector[TestEntity, int64](
			func(e *TestEntity) int64 { return e.ID },
		)

		entities := []TestEntity{
			{ID: 1, Text: "hello world test"},
			{ID: 2, Text: "hello beautiful world"},
			{ID: 3, Text: "goodbye world"},
		}
		repo.BatchCreate(ctx, entities)

		filter := NewFilter().
			Where("text", OpLike, "%world%").
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}
	})

	t.Run("OpILike case-insensitive", func(t *testing.T) {
		type TestEntity struct {
			ID   int64  `db:"id"`
			Name string `db:"name"`
		}

		repo := NewInMemoryConnector[TestEntity, int64](
			func(e *TestEntity) int64 { return e.ID },
		)

		entities := []TestEntity{
			{ID: 1, Name: "HELLO"},
			{ID: 2, Name: "hello"},
			{ID: 3, Name: "HeLLo"},
			{ID: 4, Name: "goodbye"},
		}
		repo.BatchCreate(ctx, entities)

		filter := NewFilter().
			Where("name", OpILike, "hello").
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}
	})

	t.Run("OpLike no matches", func(t *testing.T) {
		type TestEntity struct {
			ID   int64  `db:"id"`
			Text string `db:"text"`
		}

		repo := NewInMemoryConnector[TestEntity, int64](
			func(e *TestEntity) int64 { return e.ID },
		)

		entities := []TestEntity{
			{ID: 1, Text: "test"},
		}
		repo.BatchCreate(ctx, entities)

		filter := NewFilter().
			Where("text", OpLike, "%nomatch%").
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("Expected 0 results, got %d", len(results))
		}
	})
}

func TestInMemoryCompareEdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("Compare float values", func(t *testing.T) {
		type TestEntity struct {
			ID    int64   `db:"id"`
			Price float64 `db:"price"`
		}

		repo := NewInMemoryConnector[TestEntity, int64](
			func(e *TestEntity) int64 { return e.ID },
		)

		entities := []TestEntity{
			{ID: 1, Price: 10.5},
			{ID: 2, Price: 20.75},
			{ID: 3, Price: 15.0},
		}
		repo.BatchCreate(ctx, entities)

		filter := NewFilter().
			Where("price", OpGreaterThan, 15.0).
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}
		if len(results) > 0 && results[0].Price != 20.75 {
			t.Errorf("Expected price 20.75, got %f", results[0].Price)
		}
	})

	t.Run("Compare uint values", func(t *testing.T) {
		type TestEntity struct {
			ID    int64 `db:"id"`
			Count uint  `db:"count"`
		}

		repo := NewInMemoryConnector[TestEntity, int64](
			func(e *TestEntity) int64 { return e.ID },
		)

		entities := []TestEntity{
			{ID: 1, Count: 5},
			{ID: 2, Count: 10},
			{ID: 3, Count: 15},
		}
		repo.BatchCreate(ctx, entities)

		filter := NewFilter().
			Where("count", OpLessThan, uint(12)).
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}
	})

	t.Run("Compare int32 values", func(t *testing.T) {
		type TestEntity struct {
			ID  int64 `db:"id"`
			Val int32 `db:"val"`
		}

		repo := NewInMemoryConnector[TestEntity, int64](
			func(e *TestEntity) int64 { return e.ID },
		)

		entities := []TestEntity{
			{ID: 1, Val: 100},
			{ID: 2, Val: 200},
		}
		repo.BatchCreate(ctx, entities)

		filter := NewFilter().
			Where("val", OpEqual, int32(100)).
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}
	})

	t.Run("Sort by different types", func(t *testing.T) {
		type TestEntity struct {
			ID       int64   `db:"id"`
			IntVal   int     `db:"int_val"`
			FloatVal float64 `db:"float_val"`
			StrVal   string  `db:"str_val"`
		}

		repo := NewInMemoryConnector[TestEntity, int64](
			func(e *TestEntity) int64 { return e.ID },
		)

		entities := []TestEntity{
			{ID: 1, IntVal: 30, FloatVal: 3.5, StrVal: "z"},
			{ID: 2, IntVal: 10, FloatVal: 1.5, StrVal: "a"},
			{ID: 3, IntVal: 20, FloatVal: 2.5, StrVal: "m"},
		}
		repo.BatchCreate(ctx, entities)

		// Sort by int - use capital field names for reflection
		filter := NewFilter().
			OrderBy("IntVal", SortAsc).
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if results[0].IntVal != 10 || results[1].IntVal != 20 || results[2].IntVal != 30 {
			t.Errorf("Int sort failed: %v", results)
		}

		// Sort by float
		filter = NewFilter().
			OrderBy("FloatVal", SortDesc).
			Build()

		results, err = repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if results[0].FloatVal != 3.5 || results[1].FloatVal != 2.5 || results[2].FloatVal != 1.5 {
			t.Errorf("Float sort failed: %v", results)
		}

		// Sort by string
		filter = NewFilter().
			OrderBy("StrVal", SortAsc).
			Build()

		results, err = repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if results[0].StrVal != "a" || results[1].StrVal != "m" || results[2].StrVal != "z" {
			t.Errorf("String sort failed: %v", results)
		}
	})
}
