package sietch

import (
	"context"
	"testing"

	"github.com/seb7887/gofw/sietch/internal/testutils"
)

func TestInMemoryAdvancedOperators(t *testing.T) {
	ctx := context.Background()

	t.Run("OpIn operator", func(t *testing.T) {
		repo := NewInMemoryConnector[testutils.Account, int64](
			func(a *testutils.Account) int64 { return a.ID },
		)

		accounts := []testutils.Account{
			{ID: 1, Balance: 100},
			{ID: 2, Balance: 200},
			{ID: 3, Balance: 300},
			{ID: 4, Balance: 400},
		}
		repo.BatchCreate(ctx, accounts)

		filter := NewFilter().
			Where("balance", OpIn, []int{100, 300}).
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}
	})

	t.Run("OpNotIn operator", func(t *testing.T) {
		repo := NewInMemoryConnector[testutils.Account, int64](
			func(a *testutils.Account) int64 { return a.ID },
		)

		accounts := []testutils.Account{
			{ID: 1, Balance: 100},
			{ID: 2, Balance: 200},
			{ID: 3, Balance: 300},
		}
		repo.BatchCreate(ctx, accounts)

		filter := NewFilter().
			Where("balance", OpNotIn, []int{100, 300}).
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}
		if len(results) > 0 && results[0].Balance != 200 {
			t.Errorf("Expected balance 200, got %d", results[0].Balance)
		}
	})

	t.Run("OpBetween operator", func(t *testing.T) {
		repo := NewInMemoryConnector[testutils.Account, int64](
			func(a *testutils.Account) int64 { return a.ID },
		)

		accounts := []testutils.Account{
			{ID: 1, Balance: 100},
			{ID: 2, Balance: 200},
			{ID: 3, Balance: 300},
			{ID: 4, Balance: 400},
		}
		repo.BatchCreate(ctx, accounts)

		filter := NewFilter().
			Where("balance", OpBetween, []int{150, 350}).
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}
	})

	t.Run("OpIsNull operator", func(t *testing.T) {
		type TestEntity struct {
			ID    int64  `db:"id"`
			Value string `db:"value"`
		}

		repo := NewInMemoryConnector[TestEntity, int64](
			func(e *TestEntity) int64 { return e.ID },
		)

		entities := []TestEntity{
			{ID: 1, Value: ""},
			{ID: 2, Value: "test"},
		}
		repo.BatchCreate(ctx, entities)

		filter := NewFilter().
			Where("value", OpIsNull, nil).
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result (empty string is zero value), got %d", len(results))
		}
	})

	t.Run("OpIsNotNull operator", func(t *testing.T) {
		type TestEntity struct {
			ID    int64  `db:"id"`
			Value string `db:"value"`
		}

		repo := NewInMemoryConnector[TestEntity, int64](
			func(e *TestEntity) int64 { return e.ID },
		)

		entities := []TestEntity{
			{ID: 1, Value: ""},
			{ID: 2, Value: "test"},
		}
		repo.BatchCreate(ctx, entities)

		filter := NewFilter().
			Where("value", OpIsNotNull, nil).
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result (non-empty string), got %d", len(results))
		}
	})
}

func TestInMemorySorting(t *testing.T) {
	ctx := context.Background()

	t.Run("Sort ascending", func(t *testing.T) {
		repo := NewInMemoryConnector[testutils.Account, int64](
			func(a *testutils.Account) int64 { return a.ID },
		)

		accounts := []testutils.Account{
			{ID: 3, Balance: 300},
			{ID: 1, Balance: 100},
			{ID: 2, Balance: 200},
		}
		repo.BatchCreate(ctx, accounts)

		filter := NewFilter().
			OrderBy("balance", SortAsc).
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 3 {
			t.Fatalf("Expected 3 results, got %d", len(results))
		}

		if results[0].Balance != 100 || results[1].Balance != 200 || results[2].Balance != 300 {
			t.Errorf("Results not sorted correctly: %v", results)
		}
	})

	t.Run("Sort descending", func(t *testing.T) {
		repo := NewInMemoryConnector[testutils.Account, int64](
			func(a *testutils.Account) int64 { return a.ID },
		)

		accounts := []testutils.Account{
			{ID: 1, Balance: 100},
			{ID: 3, Balance: 300},
			{ID: 2, Balance: 200},
		}
		repo.BatchCreate(ctx, accounts)

		filter := NewFilter().
			OrderBy("balance", SortDesc).
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 3 {
			t.Fatalf("Expected 3 results, got %d", len(results))
		}

		if results[0].Balance != 300 || results[1].Balance != 200 || results[2].Balance != 100 {
			t.Errorf("Results not sorted correctly: %v", results)
		}
	})
}

func TestInMemoryPagination(t *testing.T) {
	ctx := context.Background()

	t.Run("Limit without offset", func(t *testing.T) {
		repo := NewInMemoryConnector[testutils.Account, int64](
			func(a *testutils.Account) int64 { return a.ID },
		)

		accounts := []testutils.Account{
			{ID: 1, Balance: 100},
			{ID: 2, Balance: 200},
			{ID: 3, Balance: 300},
			{ID: 4, Balance: 400},
			{ID: 5, Balance: 500},
		}
		repo.BatchCreate(ctx, accounts)

		filter := NewFilter().Limit(3).Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}
	})

	t.Run("Offset without limit", func(t *testing.T) {
		repo := NewInMemoryConnector[testutils.Account, int64](
			func(a *testutils.Account) int64 { return a.ID },
		)

		accounts := []testutils.Account{
			{ID: 1, Balance: 100},
			{ID: 2, Balance: 200},
			{ID: 3, Balance: 300},
		}
		repo.BatchCreate(ctx, accounts)

		filter := NewFilter().Offset(1).Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}
	})

	t.Run("Limit with offset", func(t *testing.T) {
		repo := NewInMemoryConnector[testutils.Account, int64](
			func(a *testutils.Account) int64 { return a.ID },
		)

		accounts := []testutils.Account{
			{ID: 1, Balance: 100},
			{ID: 2, Balance: 200},
			{ID: 3, Balance: 300},
			{ID: 4, Balance: 400},
			{ID: 5, Balance: 500},
		}
		repo.BatchCreate(ctx, accounts)

		filter := NewFilter().
			OrderBy("balance", SortAsc).
			Offset(1).
			Limit(2).
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}
		if results[0].Balance != 200 || results[1].Balance != 300 {
			t.Errorf("Wrong results: %v", results)
		}
	})

	t.Run("Offset larger than result set", func(t *testing.T) {
		repo := NewInMemoryConnector[testutils.Account, int64](
			func(a *testutils.Account) int64 { return a.ID },
		)

		accounts := []testutils.Account{
			{ID: 1, Balance: 100},
			{ID: 2, Balance: 200},
		}
		repo.BatchCreate(ctx, accounts)

		filter := NewFilter().Offset(10).Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("Expected 0 results, got %d", len(results))
		}
	})
}

func TestInMemoryDistinct(t *testing.T) {
	ctx := context.Background()

	t.Run("Distinct removes duplicates", func(t *testing.T) {
		repo := NewInMemoryConnector[testutils.Account, int64](
			func(a *testutils.Account) int64 { return a.ID },
		)

		accounts := []testutils.Account{
			{ID: 1, Balance: 100},
			{ID: 2, Balance: 100},
			{ID: 3, Balance: 200},
		}
		repo.BatchCreate(ctx, accounts)

		filter := NewFilter().Distinct().Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		// Distinct works on full struct, so all should be unique
		if len(results) != 3 {
			t.Errorf("Expected 3 unique results, got %d", len(results))
		}
	})
}

func TestInMemoryCount(t *testing.T) {
	ctx := context.Background()

	t.Run("Count all items", func(t *testing.T) {
		repo := NewInMemoryConnector[testutils.Account, int64](
			func(a *testutils.Account) int64 { return a.ID },
		)

		accounts := []testutils.Account{
			{ID: 1, Balance: 100},
			{ID: 2, Balance: 200},
			{ID: 3, Balance: 300},
		}
		repo.BatchCreate(ctx, accounts)

		filter := &Filter{}
		count, err := repo.Count(ctx, filter)
		if err != nil {
			t.Fatalf("Count failed: %v", err)
		}

		if count != 3 {
			t.Errorf("Expected count 3, got %d", count)
		}
	})

	t.Run("Count with filter", func(t *testing.T) {
		repo := NewInMemoryConnector[testutils.Account, int64](
			func(a *testutils.Account) int64 { return a.ID },
		)

		accounts := []testutils.Account{
			{ID: 1, Balance: 100},
			{ID: 2, Balance: 200},
			{ID: 3, Balance: 300},
		}
		repo.BatchCreate(ctx, accounts)

		filter := NewFilter().
			Where("balance", OpGreaterThan, 150).
			Build()

		count, err := repo.Count(ctx, filter)
		if err != nil {
			t.Fatalf("Count failed: %v", err)
		}

		if count != 2 {
			t.Errorf("Expected count 2, got %d", count)
		}
	})

	t.Run("Count empty result", func(t *testing.T) {
		repo := NewInMemoryConnector[testutils.Account, int64](
			func(a *testutils.Account) int64 { return a.ID },
		)

		filter := NewFilter().
			Where("balance", OpGreaterThan, 1000).
			Build()

		count, err := repo.Count(ctx, filter)
		if err != nil {
			t.Fatalf("Count failed: %v", err)
		}

		if count != 0 {
			t.Errorf("Expected count 0, got %d", count)
		}
	})
}

func TestInMemoryTransactions(t *testing.T) {
	ctx := context.Background()

	t.Run("Transaction commits on success", func(t *testing.T) {
		repo := NewInMemoryConnector[testutils.Account, int64](
			func(a *testutils.Account) int64 { return a.ID },
		)

		account := testutils.Account{ID: 1, Balance: 100}
		repo.Create(ctx, &account)

		err := repo.WithTx(ctx, func(txRepo Repository[testutils.Account, int64]) error {
			acc, _ := txRepo.Get(ctx, 1)
			acc.Balance = 200
			return txRepo.Update(ctx, acc)
		})

		if err != nil {
			t.Fatalf("Transaction failed: %v", err)
		}

		updated, _ := repo.Get(ctx, 1)
		if updated.Balance != 200 {
			t.Errorf("Expected balance 200, got %d", updated.Balance)
		}
	})

	t.Run("Transaction rolls back on error", func(t *testing.T) {
		repo := NewInMemoryConnector[testutils.Account, int64](
			func(a *testutils.Account) int64 { return a.ID },
		)

		account := testutils.Account{ID: 1, Balance: 100}
		repo.Create(ctx, &account)

		err := repo.WithTx(ctx, func(txRepo Repository[testutils.Account, int64]) error {
			acc, _ := txRepo.Get(ctx, 1)
			acc.Balance = 200
			txRepo.Update(ctx, acc)
			return ErrItemNotFound // Simulate error
		})

		if err == nil {
			t.Fatal("Expected error from transaction")
		}

		// Should be rolled back to original value
		original, _ := repo.Get(ctx, 1)
		if original.Balance != 100 {
			t.Errorf("Expected balance 100 (rolled back), got %d", original.Balance)
		}
	})

	t.Run("Transaction rolls back on panic", func(t *testing.T) {
		repo := NewInMemoryConnector[testutils.Account, int64](
			func(a *testutils.Account) int64 { return a.ID },
		)

		account := testutils.Account{ID: 1, Balance: 100}
		repo.Create(ctx, &account)

		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic to be re-raised")
			}
		}()

		repo.WithTx(ctx, func(txRepo Repository[testutils.Account, int64]) error {
			acc, _ := txRepo.Get(ctx, 1)
			acc.Balance = 200
			txRepo.Update(ctx, acc)
			panic("test panic")
		})
	})
}

func TestInMemoryComplexQueries(t *testing.T) {
	ctx := context.Background()

	t.Run("Multiple conditions with sorting and pagination", func(t *testing.T) {
		repo := NewInMemoryConnector[testutils.Account, int64](
			func(a *testutils.Account) int64 { return a.ID },
		)

		accounts := []testutils.Account{
			{ID: 1, Balance: 100},
			{ID: 2, Balance: 200},
			{ID: 3, Balance: 300},
			{ID: 4, Balance: 400},
			{ID: 5, Balance: 500},
		}
		repo.BatchCreate(ctx, accounts)

		filter := NewFilter().
			Where("balance", OpGreaterThan, 150).
			OrderBy("balance", SortDesc).
			Limit(2).
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}
		if results[0].Balance != 500 || results[1].Balance != 400 {
			t.Errorf("Wrong results: %v", results)
		}
	})
}
