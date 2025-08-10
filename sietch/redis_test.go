package sietch

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/seb7887/gofw/sietch/internal/testutils"
)

func setupRedisTest(t *testing.T) (*redis.Client, *RedisConnector[testutils.Account, int64]) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // Use test database
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available for testing:", err)
	}

	// Clear test database
	client.FlushDB(ctx)

	keyFunc := func(id int64) string {
		return "account:" + string(rune(id+'0'))
	}

	connector := NewRedisConnector[testutils.Account, int64](
		client,
		5*time.Minute,
		func(a *testutils.Account) int64 { return a.ID },
		keyFunc,
	)

	return client, connector
}

func TestRedisConnector_CreateGet(t *testing.T) {
	client, repo := setupRedisTest(t)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test Create with valid account
	account := testutils.Account{ID: 1, Balance: 100}
	err := repo.Create(ctx, &account)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Test Get existing account
	acc, err := repo.Get(ctx, 1)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if acc.Balance != 100 || acc.ID != 1 {
		t.Errorf("expected ID=1, Balance=100, got ID=%d, Balance=%d", acc.ID, acc.Balance)
	}

	// Test Get non-existing account
	_, err = repo.Get(ctx, 999)
	if err != ErrItemNotFound {
		t.Errorf("expected ErrItemNotFound, got: %v", err)
	}
}

func TestRedisConnector_CreateValidation(t *testing.T) {
	_, repo := setupRedisTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test Create with nil item
	err := repo.Create(ctx, nil)
	if err == nil || err.Error() != "item cannot be nil" {
		t.Errorf("expected 'item cannot be nil' error, got: %v", err)
	}
}

func TestRedisConnector_BatchCreate(t *testing.T) {
	client, repo := setupRedisTest(t)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test BatchCreate with valid accounts
	accounts := []testutils.Account{
		{ID: 2, Balance: 200},
		{ID: 3, Balance: 300},
	}

	err := repo.BatchCreate(ctx, accounts)
	if err != nil {
		t.Fatalf("BatchCreate failed: %v", err)
	}

	// Verify accounts were created
	acc2, err := repo.Get(ctx, 2)
	if err != nil {
		t.Fatalf("Get account 2 failed: %v", err)
	}
	if acc2.Balance != 200 {
		t.Errorf("expected balance 200, got %d", acc2.Balance)
	}

	acc3, err := repo.Get(ctx, 3)
	if err != nil {
		t.Fatalf("Get account 3 failed: %v", err)
	}
	if acc3.Balance != 300 {
		t.Errorf("expected balance 300, got %d", acc3.Balance)
	}
}

func TestRedisConnector_BatchCreateValidation(t *testing.T) {
	_, repo := setupRedisTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test BatchCreate with empty slice
	err := repo.BatchCreate(ctx, []testutils.Account{})
	if err != nil {
		t.Errorf("expected no error for empty slice, got: %v", err)
	}

	// Test BatchCreate with nil slice
	err = repo.BatchCreate(ctx, nil)
	if err != nil {
		t.Errorf("expected no error for nil slice, got: %v", err)
	}
}

func TestRedisConnector_Update(t *testing.T) {
	client, repo := setupRedisTest(t)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Create initial account
	account := testutils.Account{ID: 4, Balance: 400}
	err := repo.Create(ctx, &account)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update account
	updated := testutils.Account{ID: 4, Balance: 450}
	err = repo.Update(ctx, &updated)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	acc, err := repo.Get(ctx, 4)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if acc.Balance != 450 {
		t.Errorf("expected balance 450, got %d", acc.Balance)
	}
}

func TestRedisConnector_UpdateValidation(t *testing.T) {
	_, repo := setupRedisTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test Update with nil item
	err := repo.Update(ctx, nil)
	if err == nil || err.Error() != "item cannot be nil" {
		t.Errorf("expected 'item cannot be nil' error, got: %v", err)
	}
}

func TestRedisConnector_BatchUpdate(t *testing.T) {
	client, repo := setupRedisTest(t)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Create initial accounts
	accounts := []testutils.Account{
		{ID: 5, Balance: 500},
		{ID: 6, Balance: 600},
	}
	err := repo.BatchCreate(ctx, accounts)
	if err != nil {
		t.Fatalf("BatchCreate failed: %v", err)
	}

	// Update accounts
	updates := []testutils.Account{
		{ID: 5, Balance: 550},
		{ID: 6, Balance: 650},
	}
	err = repo.BatchUpdate(ctx, updates)
	if err != nil {
		t.Fatalf("BatchUpdate failed: %v", err)
	}

	// Verify updates
	acc5, err := repo.Get(ctx, 5)
	if err != nil {
		t.Fatalf("Get account 5 failed: %v", err)
	}
	if acc5.Balance != 550 {
		t.Errorf("expected balance 550, got %d", acc5.Balance)
	}

	acc6, err := repo.Get(ctx, 6)
	if err != nil {
		t.Fatalf("Get account 6 failed: %v", err)
	}
	if acc6.Balance != 650 {
		t.Errorf("expected balance 650, got %d", acc6.Balance)
	}
}

func TestRedisConnector_BatchUpdateValidation(t *testing.T) {
	_, repo := setupRedisTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test BatchUpdate with empty slice
	err := repo.BatchUpdate(ctx, []testutils.Account{})
	if err != nil {
		t.Errorf("expected no error for empty slice, got: %v", err)
	}
}

func TestRedisConnector_Delete(t *testing.T) {
	client, repo := setupRedisTest(t)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Create account
	account := testutils.Account{ID: 7, Balance: 700}
	err := repo.Create(ctx, &account)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Delete account
	err = repo.Delete(ctx, 7)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	_, err = repo.Get(ctx, 7)
	if err != ErrItemNotFound {
		t.Errorf("expected ErrItemNotFound, got: %v", err)
	}
}

func TestRedisConnector_BatchDelete(t *testing.T) {
	client, repo := setupRedisTest(t)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Create accounts
	accounts := []testutils.Account{
		{ID: 8, Balance: 800},
		{ID: 9, Balance: 900},
	}
	err := repo.BatchCreate(ctx, accounts)
	if err != nil {
		t.Fatalf("BatchCreate failed: %v", err)
	}

	// Delete accounts
	err = repo.BatchDelete(ctx, []int64{8, 9})
	if err != nil {
		t.Fatalf("BatchDelete failed: %v", err)
	}

	// Verify deletions
	_, err = repo.Get(ctx, 8)
	if err != ErrItemNotFound {
		t.Errorf("expected ErrItemNotFound for ID 8, got: %v", err)
	}

	_, err = repo.Get(ctx, 9)
	if err != ErrItemNotFound {
		t.Errorf("expected ErrItemNotFound for ID 9, got: %v", err)
	}
}

func TestRedisConnector_BatchDeleteValidation(t *testing.T) {
	_, repo := setupRedisTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test BatchDelete with empty slice
	err := repo.BatchDelete(ctx, []int64{})
	if err != nil {
		t.Errorf("expected no error for empty slice, got: %v", err)
	}
}

func TestRedisConnector_Query(t *testing.T) {
	_, repo := setupRedisTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Query should return unsupported operation error
	filter := Filter{}
	result, err := repo.Query(ctx, &filter)
	if err != ErrUnsupportedOperation {
		t.Errorf("expected ErrUnsupportedOperation, got: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got: %v", result)
	}
}

func TestRedisConnector_TTL(t *testing.T) {
	client, _ := setupRedisTest(t)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Create connector with short TTL
	keyFunc := func(id int64) string {
		return "ttl_test:" + string(rune(id+'0'))
	}

	repo := NewRedisConnector[testutils.Account, int64](
		client,
		1*time.Second, // Short TTL for testing
		func(a *testutils.Account) int64 { return a.ID },
		keyFunc,
	)

	// Create account
	account := testutils.Account{ID: 10, Balance: 1000}
	err := repo.Create(ctx, &account)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify TTL is set
	ttl := client.TTL(ctx, keyFunc(10))
	if ttl.Val() <= 0 {
		t.Errorf("expected positive TTL, got: %v", ttl.Val())
	}
	if ttl.Val() > 1*time.Second {
		t.Errorf("expected TTL <= 1 second, got: %v", ttl.Val())
	}
}