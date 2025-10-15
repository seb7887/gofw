package sietch

import (
	"context"
	"testing"

	"github.com/seb7887/gofw/sietch/internal/testutils"
)

func TestFilterBuilder_OrConditions(t *testing.T) {
	ctx := context.Background()

	t.Run("OR with two leaf conditions", func(t *testing.T) {
		repo := NewInMemoryConnector[testutils.Account, int64](
			func(a *testutils.Account) int64 { return a.ID },
		)

		accounts := []testutils.Account{
			{ID: 1, Balance: 100},
			{ID: 2, Balance: 200},
			{ID: 3, Balance: 300},
		}
		repo.BatchCreate(ctx, accounts)

		// Query: WHERE balance = 100 OR balance = 300
		filter := NewFilter().
			Or(
				Condition{Field: "balance", Operator: OpEqual, Value: 100},
				Condition{Field: "balance", Operator: OpEqual, Value: 300},
			).
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}

		// Verify we got the right items
		found100 := false
		found300 := false
		for _, r := range results {
			if r.Balance == 100 {
				found100 = true
			}
			if r.Balance == 300 {
				found300 = true
			}
		}
		if !found100 || !found300 {
			t.Error("Expected to find balances 100 and 300")
		}
	})

	t.Run("AND + OR combination", func(t *testing.T) {
		repo := NewInMemoryConnector[testutils.Account, int64](
			func(a *testutils.Account) int64 { return a.ID },
		)

		accounts := []testutils.Account{
			{ID: 1, Balance: 50},
			{ID: 2, Balance: 150},
			{ID: 3, Balance: 250},
			{ID: 4, Balance: 350},
		}
		repo.BatchCreate(ctx, accounts)

		// Query: WHERE balance > 100 AND (balance = 150 OR balance = 350)
		filter := NewFilter().
			Where("balance", OpGreaterThan, 100).
			Or(
				Condition{Field: "balance", Operator: OpEqual, Value: 150},
				Condition{Field: "balance", Operator: OpEqual, Value: 350},
			).
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}

		for _, r := range results {
			if r.Balance != 150 && r.Balance != 350 {
				t.Errorf("Unexpected balance: %d", r.Balance)
			}
		}
	})

	t.Run("Nested OR conditions", func(t *testing.T) {
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

		// Query: WHERE (balance < 150) OR (balance > 350)
		filter := NewFilter().
			Or(
				Condition{Field: "balance", Operator: OpLessThan, Value: 150},
				Condition{Field: "balance", Operator: OpGreaterThan, Value: 350},
			).
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}

		for _, r := range results {
			if r.Balance != 100 && r.Balance != 400 {
				t.Errorf("Unexpected balance: %d", r.Balance)
			}
		}
	})
}

func TestFilterBuilder_NotConditions(t *testing.T) {
	ctx := context.Background()

	t.Run("NOT with simple condition", func(t *testing.T) {
		repo := NewInMemoryConnector[testutils.Account, int64](
			func(a *testutils.Account) int64 { return a.ID },
		)

		accounts := []testutils.Account{
			{ID: 1, Balance: 100},
			{ID: 2, Balance: 200},
			{ID: 3, Balance: 300},
		}
		repo.BatchCreate(ctx, accounts)

		// Query: WHERE NOT (balance = 200)
		filter := NewFilter().
			Not(Condition{Field: "balance", Operator: OpEqual, Value: 200}).
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}

		for _, r := range results {
			if r.Balance == 200 {
				t.Error("Should not have found balance 200")
			}
		}
	})

	t.Run("NOT with OR condition", func(t *testing.T) {
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

		// Query: WHERE NOT (balance = 100 OR balance = 400)
		filter := NewFilter().
			Not(Condition{
				LogicalOp: LogicalOR,
				Conditions: []Condition{
					{Field: "balance", Operator: OpEqual, Value: 100},
					{Field: "balance", Operator: OpEqual, Value: 400},
				},
			}).
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}

		for _, r := range results {
			if r.Balance == 100 || r.Balance == 400 {
				t.Errorf("Should not have found balance %d", r.Balance)
			}
		}
	})
}

func TestFilterBuilder_ComplexConditions(t *testing.T) {
	ctx := context.Background()

	t.Run("Complex nested conditions", func(t *testing.T) {
		repo := NewInMemoryConnector[testutils.Account, int64](
			func(a *testutils.Account) int64 { return a.ID },
		)

		accounts := []testutils.Account{
			{ID: 1, Balance: 50},
			{ID: 2, Balance: 150},
			{ID: 3, Balance: 250},
			{ID: 4, Balance: 350},
			{ID: 5, Balance: 450},
		}
		repo.BatchCreate(ctx, accounts)

		// Query: WHERE (balance < 100 OR balance > 400) AND NOT (balance = 50)
		filter := NewFilter().
			Or(
				Condition{Field: "balance", Operator: OpLessThan, Value: 100},
				Condition{Field: "balance", Operator: OpGreaterThan, Value: 400},
			).
			Not(Condition{Field: "balance", Operator: OpEqual, Value: 50}).
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		// Should only get balance 450 (not 50 because of NOT)
		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}

		if len(results) > 0 && results[0].Balance != 450 {
			t.Errorf("Expected balance 450, got %d", results[0].Balance)
		}
	})

	t.Run("AND inside OR", func(t *testing.T) {
		repo := NewInMemoryConnector[testutils.Account, int64](
			func(a *testutils.Account) int64 { return a.ID },
		)

		accounts := []testutils.Account{
			{ID: 1, Balance: 50},
			{ID: 2, Balance: 150},
			{ID: 3, Balance: 250},
			{ID: 4, Balance: 350},
		}
		repo.BatchCreate(ctx, accounts)

		// Query: WHERE (balance > 100 AND balance < 200) OR (balance > 300)
		filter := NewFilter().
			Or(
				Condition{
					LogicalOp: LogicalAND,
					Conditions: []Condition{
						{Field: "balance", Operator: OpGreaterThan, Value: 100},
						{Field: "balance", Operator: OpLessThan, Value: 200},
					},
				},
				Condition{Field: "balance", Operator: OpGreaterThan, Value: 300},
			).
			Build()

		results, err := repo.Query(ctx, filter)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		// Should get balances 150 and 350
		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}

		for _, r := range results {
			if r.Balance != 150 && r.Balance != 350 {
				t.Errorf("Unexpected balance: %d", r.Balance)
			}
		}
	})
}

func TestFilterBuilder_IsLeafIsComposite(t *testing.T) {
	t.Run("IsLeaf returns true for leaf condition", func(t *testing.T) {
		condition := Condition{
			Field:    "balance",
			Operator: OpEqual,
			Value:    100,
		}

		if !condition.IsLeaf() {
			t.Error("Expected IsLeaf() to be true")
		}

		if condition.IsComposite() {
			t.Error("Expected IsComposite() to be false")
		}
	})

	t.Run("IsComposite returns true for composite condition", func(t *testing.T) {
		condition := Condition{
			LogicalOp: LogicalOR,
			Conditions: []Condition{
				{Field: "balance", Operator: OpEqual, Value: 100},
				{Field: "balance", Operator: OpEqual, Value: 200},
			},
		}

		if condition.IsLeaf() {
			t.Error("Expected IsLeaf() to be false")
		}

		if !condition.IsComposite() {
			t.Error("Expected IsComposite() to be true")
		}
	})
}
