# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Sietch** is a Go library that provides a unified, generic repository interface for CRUD operations across multiple database backends. It uses Go generics (requires Go 1.23+) and reflection to enable backend-agnostic data access patterns.

### Core Architecture

The project follows a clean architecture pattern with these key components:

- **Repository Interface** (`repository.go`): Generic contract defining CRUD operations for any entity type `T` with identifier type `ID`
- **Backend Implementations**:
  - `CockroachDBConnector` (`cockroach.go`, `cockroach_tx.go`): PostgreSQL/CockroachDB implementation using pgxpool with transaction support
  - `InMemoryConnector` (`inmemory.go`, `inmemory_tx.go`): In-memory implementation with thread-safe operations and snapshot-based transactions
  - `RedisConnector` (`redis.go`): Redis-based caching implementation with TTL support (limited operations)
- **Supporting Types**:
  - `Filter`, `FilterBuilder`, `Condition` (`filters.go`): Advanced query filtering system with builder pattern
  - Custom errors (`errors.go`): Domain-specific error types

### Key Design Patterns

1. **Generic Type Constraints**: All connectors use `[T any, ID comparable]` type parameters
2. **Builder Pattern**: `FilterBuilder` provides fluent API for constructing queries
3. **Reflection-Based Mapping**: CockroachDB and InMemory connectors use reflection for field access
4. **ID Extraction Functions**: Each connector requires a `getID func(*T) ID` function to extract entity identifiers
5. **Transaction Support**:
   - CockroachDB: ACID transactions via pgx
   - InMemory: Snapshot-based transactions with rollback
   - Redis: Not supported (returns `ErrUnsupportedOperation`)
6. **Pipeline Optimization**: Redis connector uses pipelines for batch operations

## Features

### Advanced Filtering (Priority 1 Implementation)

The library now supports comprehensive filtering capabilities:

**Operators:**
- Comparison: `OpEqual`, `OpNotEqual`, `OpGreaterThan`, `OpLessThan`, `OpGreaterThanOrEqual`, `OpLessThanOrEqual`
- Advanced: `OpIn`, `OpNotIn`, `OpLike`, `OpILike`, `OpIsNull`, `OpIsNotNull`, `OpBetween`

**Sorting:**
- Multi-field sorting with `SortAsc` and `SortDesc`
- Implemented via `OrderBy()` method in FilterBuilder

**Pagination:**
- `Limit()` and `Offset()` methods
- Efficient result set limiting

**Distinct:**
- Remove duplicate results via `Distinct()` method

**Aggregations:**
- `Count()` method for efficient counting without fetching all results

**Example:**
```go
filter := sietch.NewFilter().
    Where("balance", sietch.OpGreaterThan, 100).
    Where("status", sietch.OpIn, []string{"active", "pending"}).
    OrderBy("balance", sietch.SortDesc).
    Limit(10).
    Offset(20).
    Build()

results, _ := repo.Query(ctx, filter)
count, _ := repo.Count(ctx, filter)
```

### Transactions

**CockroachDB/PostgreSQL:**
```go
txRepo, ok := repo.(sietch.Transactional[Account, int64])
err := txRepo.WithTx(ctx, func(tx sietch.Repository[Account, int64]) error {
    // Operations within transaction
    return nil // Commit or return error to rollback
})
```

**InMemory:**
- Uses snapshot/restore mechanism
- Supports panic recovery with automatic rollback

**Redis:**
- Not supported - returns `ErrUnsupportedOperation`

## Common Development Commands

### Testing
```bash
go test ./...                    # Run all tests
go test -v                       # Run tests with verbose output
go test -cover ./...             # Run tests with coverage report
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
6. Optionally implement `Transactional[T, ID]` interface for transaction support

### Entity Mapping Requirements

**For CockroachDB connector entities:**
- Use struct tags with `db:"column_name"` format
- First field with `db` tag is assumed to be the primary key
- Example: `type Account struct { ID int64 \`db:"id"\`; Balance int \`db:"balance"\` }`
- Field validation prevents SQL injection

**For InMemory connector:**
- Uses reflection with capitalized field names
- Example: Filter by "Balance" (capitalized), not "balance"
- No struct tags required

### Query Filtering

The filtering system supports:
- **Comparison Operators**: `=`, `!=`, `>`, `<`, `>=`, `<=`
- **Advanced Operators**: `IN`, `NOT IN`, `LIKE`, `ILIKE`, `IS NULL`, `IS NOT NULL`, `BETWEEN`
- **Multiple Conditions**: Combined with AND logic
- **Type-safe Comparisons**: Handles numeric types (int, int32, int64, uint, float32, float64) and strings
- **Builder Pattern**: Fluent API via `NewFilter().Where().OrderBy().Limit().Build()`

### Testing Patterns and Coverage

**Test Coverage Guidelines:**
- This is a database interaction library - **full test coverage is not expected**
- Focus testing efforts on:
  - **InMemory connector**: Can achieve high coverage (no external dependencies)
  - **Filter/Builder logic**: Business logic and query building
  - **Validation and error handling**: Field validation, operator validation
  - **Transaction interfaces**: Interface compliance tests
- Limited testing for:
  - **CockroachDB connector**: Requires actual database connection
  - **Redis connector**: Requires actual Redis instance
- **Current coverage: ~49%** is acceptable given the nature of database connectors
- When adding features, prioritize testing:
  1. InMemory implementation (testable without dependencies)
  2. Query builders and validators
  3. Interface compliance
  4. Error paths

**Test Organization:**
- `internal/testutils`: Shared test entities (e.g., `Account` struct)
- `*_test.go`: Unit tests for each component
- `cockroach_query_test.go`: Query builder tests (no DB required)
- `inmemory_*_test.go`: Comprehensive InMemory tests
- `*_tx_test.go`: Transaction interface tests

**Running Specific Tests:**
```bash
go test -run TestInMemory     # Only InMemory tests
go test -run TestFilter       # Only Filter tests
go test -cover               # With coverage
```

## Dependencies

- **pgxpool** (CockroachDB/PostgreSQL): `github.com/jackc/pgx/v5`
- **Redis client**: `github.com/redis/go-redis/v9`
- **Testing**: `github.com/stretchr/testify`

## Module Information

- Module path: `github.com/seb7887/gofw/sietch`
- Go version: 1.23.0
- Part of the larger `gofw` (Go Framework) collection

## Recent Changes (Priority 1 Implementation)

### New Features Added:
1. **Advanced Filter System**:
   - Added `FilterBuilder` with fluent API
   - 13 comparison operators including `OpIn`, `OpLike`, `OpBetween`
   - Multi-field sorting with `SortAsc`/`SortDesc`
   - Pagination via `Limit()` and `Offset()`
   - `Distinct()` for unique results

2. **Aggregations**:
   - `Count()` method added to `Repository` interface
   - Efficient counting for CockroachDB (SQL COUNT)
   - In-memory counting via filtering
   - Redis returns `ErrUnsupportedOperation`

3. **Transactions**:
   - New `Transactional[T, ID]` interface
   - `WithTx()` method with closure-based API
   - CockroachDB: Native ACID transactions
   - InMemory: Snapshot-based with panic recovery
   - Redis: Not supported

4. **Query Builder Enhancements**:
   - SQL injection prevention via field validation
   - Support for complex WHERE clauses
   - ORDER BY with multiple fields
   - DISTINCT support
   - LIMIT/OFFSET support

### Files Modified:
- `filters.go`: Complete rewrite with builder pattern
- `repository.go`: Added `Count()` and `Transactional` interface
- `cockroach.go`: Enhanced query builder, validation, Count()
- `cockroach_tx.go`: New transaction implementation
- `inmemory.go`: Added sorting, pagination, advanced operators, Count()
- `inmemory_tx.go`: New snapshot-based transactions
- `redis.go`: Added Count() and WithTx() stubs
- `README.md`: Comprehensive English documentation

### Breaking Changes:
- `queryBuilder()` in CockroachDB now returns 3 values: `(query, args, error)` instead of 2
- Filter conditions now use type-safe `ComparisonOperator` constants instead of strings
