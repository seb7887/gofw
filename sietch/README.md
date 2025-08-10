# sietch

**sietch** is a Go package that provides a unified, generic repository interface for performing CRUD operations across multiple database backends. Using Go generics and reflection, Sietch enables you to write data-access code in a backend-agnostic manner. Out-of-the-box implementations include:

- **InMemoryConnector**: Useful for testing and business logic prototyping.
- **CockroachDBConnector**: A generic implementation for CockroachDB/PostgreSQL using [pgxpool](https://github.com/jackc/pgx) that supports real CRUD operations.
- **RedisConnector**: A cache repository that serializes entities to JSON and supports setting a default TTL (time-to-live).

## Features

- **Unified CRUD Interface**: Define operations like `Create`, `Get`, `BatchCreate`, `Query`, `Update`, `BatchUpdate`, `Delete`, and `BatchDelete` in a single interface.
- **Backend Agnostic**: Write your business logic once and use dependency injection to switch between in-memory, SQL, or cache backends.
- **Generics and Reflection**: Automatically map struct fields (using `db` tags) to database columns, build SQL queries dynamically, and serialize/deserialize JSON for Redis.
- **Batch Operations**: Efficiently perform batch updates and deletes using transactions (for CockroachDB) or pipelines (for Redis).
- **Cache with TTL**: The Redis connector is designed for caching with a configurable default TTL.
- **Input Validation**: All connectors validate input parameters (nil checks, empty arrays) for robust error handling.
- **Pipeline Optimization**: Redis connector uses pipelines for batch operations and validates data before executing operations.

## Requirements

- **Go 1.18+** (for generics support)
- For CockroachDBConnector: [pgxpool](https://github.com/jackc/pgx) v5
- For RedisConnector: [go-redis/redis/v8](https://github.com/go-redis/redis) v8

## Installation

Use `go get` to add Sietch to your module:

```sh
go get github.com/seb7887/gofw/sietch
```

## Quick Start

### Define Your Entity

```go
type Account struct {
    ID      int64 `db:"id"`
    Balance int   `db:"balance"`
}
```

### Redis Connector

```go
import (
    "context"
    "time"
    "github.com/go-redis/redis/v8"
    "github.com/seb7887/gofw/sietch"
)

func main() {
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
        DB:   0,
    })

    keyFunc := func(id int64) string {
        return fmt.Sprintf("account:%d", id)
    }

    repo := sietch.NewRedisConnector[Account, int64](
        client,
        5*time.Minute, // default TTL
        func(a *Account) int64 { return a.ID },
        keyFunc,
    )

    ctx := context.Background()
    
    // Create
    account := &Account{ID: 1, Balance: 100}
    err := repo.Create(ctx, account)
    
    // Get
    acc, err := repo.Get(ctx, 1)
    
    // Batch operations
    accounts := []Account{
        {ID: 2, Balance: 200},
        {ID: 3, Balance: 300},
    }
    err = repo.BatchCreate(ctx, accounts)
}
```

### CockroachDB Connector

```go
import (
    "context"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/seb7887/gofw/sietch"
)

func main() {
    pool, _ := pgxpool.New(context.Background(), "postgres://user:pass@localhost:26257/mydb")
    
    repo := sietch.NewCockroachDBConnector[Account, int64](
        pool,
        "accounts", // table name
        func(a *Account) int64 { return a.ID },
    )

    ctx := context.Background()
    
    // Create
    account := &Account{ID: 1, Balance: 100}
    err := repo.Create(ctx, account)
    
    // Query with filters
    filter := &sietch.Filter{
        Conditions: []sietch.Condition{
            {Field: "balance", Operator: ">=", Value: 200},
        },
    }
    accounts, err := repo.Query(ctx, filter)
}
```

## Testing

The package includes comprehensive test suites for all connectors:

```sh
# Run all tests
go test ./...

# Run tests with verbose output
go test -v

# Test specific connector
go test -run TestRedisConnector
```

### Redis Tests Requirements

Redis tests require a running Redis instance on `localhost:6379`. Tests use database 1 to avoid conflicts with your data. You can start Redis using Docker:

```sh
docker run -d -p 6379:6379 redis:alpine
```

### Test Coverage

- ✅ CRUD operations validation
- ✅ Batch operations
- ✅ Input parameter validation  
- ✅ Error handling
- ✅ TTL functionality (Redis)
- ✅ Transaction support (CockroachDB)

## Error Handling

The package defines domain-specific errors:

```go
var (
    ErrItemNotFound         = errors.New("item not found")
    ErrItemAlreadyExists    = errors.New("item already exists") 
    ErrNoUpdateItem         = errors.New("no item has been updated")
    ErrNoDeleteItem         = errors.New("no item has been deleted")
    ErrUnsupportedOperation = errors.New("unsupported operation")
)
```

## Architecture

### Repository Interface

All connectors implement the generic `Repository[T, ID]` interface:

```go
type Repository[T any, ID comparable] interface {
    Create(ctx context.Context, item *T) error
    Get(ctx context.Context, id ID) (*T, error)
    BatchCreate(ctx context.Context, items []T) error
    Query(ctx context.Context, filter *Filter) ([]T, error)
    Update(ctx context.Context, item *T) error
    BatchUpdate(ctx context.Context, items []T) error
    Delete(ctx context.Context, id ID) error
    BatchDelete(ctx context.Context, items []ID) error
}
```

### Key Features by Connector

| Feature | InMemory | CockroachDB | Redis |
|---------|----------|-------------|-------|
| CRUD Operations | ✅ | ✅ | ✅ |
| Batch Operations | ✅ | ✅ | ✅ |
| Query/Filtering | ✅ | ✅ | ❌ |
| Transactions | ❌ | ✅ | ❌ |
| Pipelines | ❌ | ❌ | ✅ |
| TTL Support | ❌ | ❌ | ✅ |
| Input Validation | ✅ | ✅ | ✅ |

## Contributing

This package is part of the `gofw` (Go Framework) collection. Contributions are welcome!

## License

MIT License - see LICENSE file for details.
