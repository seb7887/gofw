# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Sietch** is a Go library that provides a unified, generic repository interface for CRUD operations across multiple database backends. It uses Go generics (requires Go 1.18+) and reflection to enable backend-agnostic data access patterns.

### Core Architecture

The project follows a clean architecture pattern with these key components:

- **Repository Interface** (`repository.go`): Generic contract defining CRUD operations for any entity type `T` with identifier type `ID`
- **Backend Implementations**:
  - `CockroachDBConnector` (`cockroach.go`): PostgreSQL/CockroachDB implementation using pgxpool
  - `InMemoryConnector` (`inmemory.go`): In-memory implementation with thread-safe operations
  - `RedisConnector` (`redis.go`): Redis-based caching implementation with TTL support
- **Supporting Types**:
  - `Filter` and `Condition` (`filters.go`): Query filtering system
  - Custom errors (`errors.go`): Domain-specific error types

### Key Design Patterns

1. **Generic Type Constraints**: All connectors use `[T any, ID comparable]` type parameters
2. **Reflection-Based Mapping**: CockroachDB connector uses struct field tags (`db:"column_name"`) for column mapping
3. **ID Extraction Functions**: Each connector requires a `getID func(*T) ID` function to extract entity identifiers
4. **Transaction Support**: CockroachDB connector uses transactions for batch operations
5. **Pipeline Optimization**: Redis connector uses pipelines for batch operations

## Common Development Commands

### Testing
```bash
go test ./...                    # Run all tests
go test -v                       # Run tests with verbose output
go test ./internal/testutils     # Test utilities package
```

### Building
```bash
go build                         # Build the module
go mod tidy                      # Clean up dependencies
go mod download                  # Download dependencies
```

### Code Quality
```bash
go fmt ./...                     # Format code
go vet ./...                     # Static analysis
```

## Working with the Codebase

### Adding New Backend Implementations

When implementing a new repository backend:

1. Implement the `Repository[T, ID]` interface from `repository.go`
2. Follow the constructor pattern: `func NewXConnector[T any, ID comparable](...) *XConnector[T, ID]`
3. Handle the `getID func(*T) ID` parameter for entity identification
4. Consider batch operation optimization (transactions, pipelines, etc.)
5. Add appropriate error handling using the predefined error types from `errors.go`

### Entity Mapping Requirements

For CockroachDB connector entities:
- Use struct tags with `db:"column_name"` format
- First field with `db` tag is assumed to be the primary key
- Example: `type Account struct { ID int64 \`db:"id"\`; Balance int \`db:"balance"\` }`

### Query Filtering

The filtering system supports:
- Operators: `=`, `!=`, `>`, `<`, `>=`, `<=`
- Multiple conditions (combined with AND logic)
- Type-safe value comparisons for numeric and string types

### Testing Patterns

Tests use the `internal/testutils` package:
- `Account` struct serves as a test entity with `id` and `balance` fields
- Constructor tests verify proper initialization and column extraction
- Value extraction tests ensure reflection-based mapping works correctly

## Dependencies

- **pgxpool** (CockroachDB/PostgreSQL): `github.com/jackc/pgx/v5`
- **Redis client**: `github.com/go-redis/redis/v8`
- **Error handling**: `github.com/pkg/errors`

## Module Information

- Module path: `github.com/seb7887/gofw/sietch`
- Go version: 1.23.0
- Part of the larger `gofw` (Go Framework) collection