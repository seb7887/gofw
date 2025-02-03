package sietch

import (
	"context"
	"github.com/seb7887/gofw/sietch/internal/testutils"
	"testing"
	"time"
)

func TestInMemoryConnector_CreateGet(t *testing.T) {
	repo := NewInMemoryConnector[testutils.Account](func(a *testutils.Account) int64 { return a.ID })
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test table para Create
	tests := []struct {
		name        string
		account     testutils.Account
		expectError bool
	}{
		{"create a valid account", testutils.Account{ID: 1, Balance: 100}, false},
		{"create duplicated account", testutils.Account{ID: 1, Balance: 200}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := repo.Create(ctx, &tc.account)
			if (err != nil) != tc.expectError {
				t.Errorf("expected error %v, got: %v", tc.expectError, err)
			}
		})
	}

	acc, err := repo.Get(ctx, 1)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if acc.Balance != 100 {
		t.Errorf("expected balance 100, got %d", acc.Balance)
	}

	_, err = repo.Get(ctx, 999)
	if err == nil {
		t.Error("expected error, got none")
	}
}

func TestInMemoryConnector_BatchCreate_Query(t *testing.T) {
	repo := NewInMemoryConnector[testutils.Account](func(a *testutils.Account) int64 { return a.ID })
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	accounts := []testutils.Account{
		{ID: 2, Balance: 200},
		{ID: 3, Balance: 300},
	}

	if err := repo.BatchCreate(ctx, accounts); err != nil {
		t.Fatalf("BatchCreate failed: %v", err)
	}

	filter := Filter{
		Conditions: []Condition{
			{Field: "balance", Operator: ">=", Value: 250},
		},
	}
	result, err := repo.Query(ctx, &filter)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 account, got %d", len(result))
	}
	if result[0].ID != 3 {
		t.Errorf("expected account with ID 3, got %d", result[0].ID)
	}
}

func TestInMemoryConnector_Update_BatchUpdate_Delete_BatchDelete(t *testing.T) {
	repo := NewInMemoryConnector[testutils.Account](func(a *testutils.Account) int64 { return a.ID })
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	accounts := []testutils.Account{
		{ID: 4, Balance: 400},
		{ID: 5, Balance: 500},
	}
	if err := repo.BatchCreate(ctx, accounts); err != nil {
		t.Fatalf("BatchCreate falló: %v", err)
	}

	// Test Update
	updated := testutils.Account{ID: 4, Balance: 450}
	if err := repo.Update(ctx, &updated); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	a, _ := repo.Get(ctx, 4)
	if a.Balance != 450 {
		t.Errorf("expected balance 450, got %d", a.Balance)
	}

	// Test BatchUpdate
	updates := []testutils.Account{
		{ID: 4, Balance: 460},
		{ID: 5, Balance: 550},
	}
	if err := repo.BatchUpdate(ctx, updates); err != nil {
		t.Fatalf("BatchUpdate failed: %v", err)
	}
	a4, _ := repo.Get(ctx, 4)
	a5, _ := repo.Get(ctx, 5)
	if a4.Balance != 460 || a5.Balance != 550 {
		t.Errorf("expected balances 460 y 550, got %d y %d", a4.Balance, a5.Balance)
	}

	// Test Delete
	if err := repo.Delete(ctx, 4); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, err := repo.Get(ctx, 4)
	if err == nil {
		t.Error("expected error with ID 4")
	}

	// Test BatchDelete
	if err := repo.BatchDelete(ctx, []int64{5}); err != nil {
		t.Fatalf("BatchDelete failed: %v", err)
	}
	_, err = repo.Get(ctx, 5)
	if err == nil {
		t.Error("expected error with ID 5")
	}
}
