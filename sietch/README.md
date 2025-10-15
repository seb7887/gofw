# Sietch

[![Go Version](https://img.shields.io/badge/Go-1.23%2B-blue)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

A generic, type-safe repository pattern implementation for Go with support for multiple database backends.

## Overview

Sietch provides a unified, type-safe interface for CRUD operations across CockroachDB/PostgreSQL, Redis, and in-memory storage using Go generics.

## Features

- ✨ **Type-Safe Generics**: Compile-time type safety with Go 1.23+
- 🔍 **Advanced Filtering**: IN, LIKE, BETWEEN, IS NULL operators with fluent API
- 📊 **Sorting & Pagination**: Multi-field sorting, LIMIT, OFFSET
- 📈 **Aggregations**: Efficient Count() method
- 💾 **Transactions**: Safe ACID transactions with auto-rollback
- 🔒 **SQL Injection Protection**: Built-in field validation
- ⚡ **Batch Operations**: Optimized bulk operations

## Installation

```bash
go get github.com/seb7887/gofw/sietch
```

## Quick Start

### Define Entity

```go
type Account struct {
    ID      int64  `db:"id"`
    Balance int    `db:"balance"`
    Status  string `db:"status"`
}
```

### CockroachDB/PostgreSQL

```go
pool, _ := sietch.NewCockroachDBConnPool(ctx, "postgres://user:pass@localhost:26257/db")
repo, _ := sietch.NewCockroachDBConnector[Account, int64](
    pool, "accounts", func(a *Account) int64 { return a.ID },
)
```

### In-Memory (Testing)

```go
repo := sietch.NewInMemoryConnector[Account, int64](
    func(a *Account) int64 { return a.ID },
)
```

### Redis

```go
client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
repo := sietch.NewRedisConnector[Account, int64](
    client, 5*time.Minute,
    func(a *Account) int64 { return a.ID },
    func(id int64) string { return fmt.Sprintf("account:%d", id) },
)
```

## Basic Operations

```go
// Create
account := &Account{ID: 1, Balance: 1000, Status: "active"}
repo.Create(ctx, account)

// Get
account, _ := repo.Get(ctx, 1)

// Update
account.Balance = 1500
repo.Update(ctx, account)

// Delete
repo.Delete(ctx, 1)

// Batch
accounts := []Account{{ID: 2, Balance: 500}, {ID: 3, Balance: 750}}
repo.BatchCreate(ctx, accounts)
```

## Advanced Filtering

### Filter Builder

```go
filter := sietch.NewFilter().
    Where("balance", sietch.OpGreaterThan, 100).
    Where("status", sietch.OpEqual, "active").
    OrderBy("balance", sietch.SortDesc).
    Limit(10).
    Offset(20).
    Build()

results, _ := repo.Query(ctx, filter)
```

### Operators

```go
// Comparison
sietch.OpEqual              // =
sietch.OpNotEqual           // !=
sietch.OpGreaterThan        // >
sietch.OpLessThan           // <
sietch.OpGreaterThanOrEqual // >=
sietch.OpLessThanOrEqual    // <=

// Advanced
sietch.OpIn        // IN (value: []any)
sietch.OpNotIn     // NOT IN
sietch.OpLike      // LIKE (pattern matching)
sietch.OpILike     // ILIKE (case-insensitive)
sietch.OpIsNull    // IS NULL
sietch.OpIsNotNull // IS NOT NULL
sietch.OpBetween   // BETWEEN (value: [2]any{min, max})
```

### Examples

**IN Operator:**
```go
filter := sietch.NewFilter().
    Where("status", sietch.OpIn, []string{"active", "pending"}).
    Build()
```

**LIKE Pattern:**
```go
filter := sietch.NewFilter().
    Where("email", sietch.OpLike, "%@example.com").
    Build()
```

**BETWEEN:**
```go
filter := sietch.NewFilter().
    Where("balance", sietch.OpBetween, []int{100, 1000}).
    Build()
```

**Multi-field Sort:**
```go
filter := sietch.NewFilter().
    OrderBy("status", sietch.SortAsc).
    OrderBy("balance", sietch.SortDesc).
    Build()
```

## Aggregations

```go
// Count records
filter := sietch.NewFilter().Where("status", sietch.OpEqual, "active").Build()
count, _ := repo.Count(ctx, filter)

// Pagination with total
results, _ := repo.Query(ctx, filter)
total, _ := repo.Count(ctx, &sietch.Filter{Conditions: filter.Conditions})
totalPages := (total + pageSize - 1) / pageSize
```

## Transactions

### CockroachDB

```go
txRepo, ok := repo.(sietch.Transactional[Account, int64])
if !ok {
    return errors.New("transactions not supported")
}

err := txRepo.WithTx(ctx, func(tx sietch.Repository[Account, int64]) error {
    // Get & debit account 1
    acc1, _ := tx.Get(ctx, 1)
    acc1.Balance -= 100
    tx.Update(ctx, acc1)
    
    // Get & credit account 2
    acc2, _ := tx.Get(ctx, 2)
    acc2.Balance += 100
    tx.Update(ctx, acc2)
    
    return nil // Commit (return error to rollback)
})
```

### InMemory

Supports transactions via snapshot/restore mechanism.

### Redis

Returns `ErrUnsupportedOperation`.

## Backend Comparison

| Feature | CockroachDB | InMemory | Redis |
|---------|-------------|----------|-------|
| CRUD | ✅ | ✅ | ✅ |
| Batch Ops | ✅ Transaction | ✅ Atomic | ✅ Pipeline |
| Query/Filter | ✅ Full SQL | ✅ In-memory | ❌ |
| Advanced Ops | ✅ All | ✅ All | ❌ |
| Sorting | ✅ Database | ✅ In-memory | ❌ |
| Pagination | ✅ | ✅ | ❌ |
| Count() | ✅ Efficient | ✅ | ❌ |
| Transactions | ✅ ACID | ✅ Snapshot | ❌ |
| Use Case | Production | Testing | Cache |

## Error Handling

```go
import "errors"

account, err := repo.Get(ctx, 999)
if errors.Is(err, sietch.ErrItemNotFound) {
    // Handle not found
}

err = repo.Create(ctx, duplicate)
if errors.Is(err, sietch.ErrItemAlreadyExists) {
    // Handle duplicate
}

err = repo.Update(ctx, nonExistent)
if errors.Is(err, sietch.ErrNoUpdateItem) {
    // No rows updated
}

results, err := redisRepo.Query(ctx, filter)
if errors.Is(err, sietch.ErrUnsupportedOperation) {
    // Operation not supported
}
```

## Complete Examples

### Pagination

```go
func GetActiveAccounts(ctx context.Context, repo sietch.Repository[Account, int64], 
    page, pageSize int) ([]Account, int, error) {
    
    filter := sietch.NewFilter().
        Where("status", sietch.OpEqual, "active").
        OrderBy("created_at", sietch.SortDesc).
        Limit(pageSize).
        Offset((page - 1) * pageSize).
        Build()
    
    items, _ := repo.Query(ctx, filter)
    total, _ := repo.Count(ctx, &sietch.Filter{Conditions: filter.Conditions})
    
    return items, int(total), nil
}
```

### Search with Multiple Conditions

```go
func SearchAccounts(ctx context.Context, repo sietch.Repository[Account, int64], 
    minBalance int, statuses []string) ([]Account, error) {
    
    filter := sietch.NewFilter().
        Where("balance", sietch.OpGreaterThanOrEqual, minBalance).
        Where("status", sietch.OpIn, statuses).
        Where("deleted_at", sietch.OpIsNull, nil).
        OrderBy("balance", sietch.SortDesc).
        Build()
    
    return repo.Query(ctx, filter)
}
```

### Money Transfer with Transaction

```go
func Transfer(ctx context.Context, repo sietch.Repository[Account, int64], 
    fromID, toID int64, amount int) error {
    
    txRepo, ok := repo.(sietch.Transactional[Account, int64])
    if !ok {
        return errors.New("transactions not supported")
    }
    
    return txRepo.WithTx(ctx, func(tx sietch.Repository[Account, int64]) error {
        from, _ := tx.Get(ctx, fromID)
        if from.Balance < amount {
            return errors.New("insufficient balance")
        }
        
        from.Balance -= amount
        tx.Update(ctx, from)
        
        to, _ := tx.Get(ctx, toID)
        to.Balance += amount
        tx.Update(ctx, to)
        
        return nil
    })
}
```

## Testing

```go
func TestAccountService(t *testing.T) {
    repo := sietch.NewInMemoryConnector[Account, int64](
        func(a *Account) int64 { return a.ID },
    )
    
    service := NewAccountService(repo)
    // Your tests...
}
```

### Run Tests

```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Integration tests (requires Docker)
docker run -d -p 26257:26257 cockroachdb/cockroach:latest start-single-node --insecure
docker run -d -p 6379:6379 redis:alpine
go test ./...
```

## Best Practices

1. **Use FilterBuilder** for readable, type-safe queries
2. **Field Validation**: Use `db:"column_name"` tags for CockroachDB
3. **Transactions** for multi-step operations
4. **Pagination**: Always use LIMIT/OFFSET for large datasets
5. **Count() over len(Query())** for better performance
6. **Batch Operations** for bulk inserts/updates/deletes
7. **Error Handling**: Check sentinel errors with `errors.Is()`

## Performance Notes

### CockroachDB
- Uses prepared statements for batch ops
- Leverages database indexes
- Efficient query planning

### InMemory  
- Thread-safe with RWMutex
- O(n) queries, O(1) lookups
- Good for <10k items

### Redis
- Pipeline optimization
- TTL auto-expiration
- Key-value lookups only

## Contributing

Part of the **gofw** (Go Framework) collection. Contributions welcome!

```bash
git clone https://github.com/seb7887/gofw.git
cd gofw/sietch
go test -v ./...
```

## License

MIT License - See [LICENSE](LICENSE) file.

## Related Packages

- [eventbus](../eventbus) - Message bus abstraction
- [httpx](../httpx) - HTTP client with circuit breaker  
- [ginsrv](../ginsrv) - Gin router utilities
- [wp](../wp) - Worker pool
- [idgen](../idgen) - ID generation
- [cfgmng](../cfgmng) - Configuration management

## Support

Open an issue on [GitHub](https://github.com/seb7887/gofw/issues).
