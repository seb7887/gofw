# Sietch

[![Go Version](https://img.shields.io/badge/Go-1.23%2B-blue)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

A generic, type-safe repository pattern implementation for Go that provides a unified interface for CRUD operations across multiple database backends.

## Overview

Sietch is a lightweight abstraction layer that allows you to write database-agnostic code using Go generics. It provides a clean, consistent API for common data access patterns while supporting multiple storage backends including CockroachDB/PostgreSQL, Redis, and in-memory storage.

## Features

- ✨ **Type-Safe Generic Interface**: Leverages Go 1.18+ generics for compile-time type safety
- 🔌 **Multiple Backend Support**: CockroachDB/PostgreSQL, Redis, and in-memory implementations
- 🔍 **Advanced Filtering**: Type-safe operators including IN, LIKE, BETWEEN, IS NULL, and more
- 🏗️ **Flexible Query Building**: Fluent API with method chaining
- 📊 **Sorting & Pagination**: Multi-field sorting with LIMIT and OFFSET support
- 📈 **Aggregations**: Count() method for efficient record counting
- 💾 **Transaction Support**: Safe transaction handling with automatic rollback
- 🔒 **Field Validation**: Built-in SQL injection protection
- ⚡ **Batch Operations**: Optimized batch create, update, and delete
- 🎯 **Zero Dependencies** (core): Only backend-specific dependencies where needed

## Installation

```bash
go get github.com/seb7887/gofw/sietch
```

**Requirements:**
- Go 1.23.0 or higher
- pgxpool v5 (for CockroachDB connector)
- go-redis/redis v8 (for Redis connector)

## Quick Start

### Define Your Entity

```go
type Account struct {
    ID      int64  `db:"id"`
    Balance int    `db:"balance"`
    Status  string `db:"status"`
}
```

### Create a Repository

#### CockroachDB/PostgreSQL

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
