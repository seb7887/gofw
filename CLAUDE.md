# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**gofw** (Go Framework) is a collection of independent Go packages that provide reusable utilities and abstractions for building Go applications. Each package is a standalone module with its own go.mod file, allowing selective imports.

### Repository Structure

This is a multi-module repository with the following packages:

- **sietch**: Generic repository interface for CRUD operations across multiple database backends (CockroachDB/PostgreSQL, Redis, in-memory)
- **eventbus**: Message bus abstraction supporting in-memory and NATS implementations
- **httpx**: HTTP client with circuit breaker (Hystrix), retry logic, and plugin system
- **ginsrv**: Gin router setup utilities with simplified route definitions and middleware
- **wp**: Worker pool implementation for parallel task execution with consistent hashing
- **idgen**: ID generation utilities (UUID and ULID)
- **cfgmng**: Configuration manager using Viper for YAML-based configs

### Go Version Requirements

- Most packages require Go 1.23.0+
- Some packages (eventbus, httpx, wp) require Go 1.24.0+
- All packages use Go generics

## Common Development Commands

### Testing

```bash
# Run all tests across all packages
go test ./...

# Run tests for a specific package
cd sietch && go test ./...
cd eventbus && go test ./...

# Run tests with verbose output
go test -v ./...

# Run a specific test
go test -run TestRedisConnector ./sietch
```

### Building and Dependencies

```bash
# Download all dependencies for all modules
go work sync  # if using Go workspace
# OR manually for each package
cd <package> && go mod download

# Clean up dependencies for a package
cd <package> && go mod tidy

# Verify all modules
go work verify  # if using Go workspace
```

### Code Quality

```bash
# Format all code
go fmt ./...

# Run static analysis
go vet ./...

# Check for common issues across all packages
for dir in sietch eventbus httpx ginsrv wp idgen cfgmng; do
  cd $dir && go vet ./... && cd ..
done
```

## Working with Multi-Module Structure

Each package is an independent Go module. When making changes:

1. Navigate to the specific package directory
2. Make changes and test within that package
3. Update the package's go.mod if adding dependencies
4. Consider impact on dependent packages (especially if changing public APIs)

### Package Dependencies

- **sietch** depends on: pgxpool, go-redis, stretchr/testify
- **eventbus** depends on: nats-io/nats.go
- **httpx** depends on: afex/hystrix-go, uber/multierr
- **ginsrv** depends on: gin-gonic/gin
- **wp** depends on: segmentio/fasthash
- **idgen** depends on: google/uuid, oklog/ulid
- **cfgmng** depends on: spf13/viper

## Architecture Patterns

### Generic Type Constraints

Most packages use Go generics for type safety:

- **sietch**: `Repository[T any, ID comparable]` interface
- **eventbus**: Generic message types with `Message` interface
- **cfgmng**: `LoadConfig[T any]` for any config struct

### Plugin/Middleware Systems

Several packages support extensibility through plugins/middleware:

- **httpx**: Plugin interface with `OnRequestStart`, `OnRequestEnd`, `OnError` callbacks
- **ginsrv**: Standard Gin middleware support
- **httpx/hystrix**: Wraps base httpx client with circuit breaker capabilities

### Worker Pool Pattern (wp)

The worker pool uses consistent hashing (FNV-1a) to distribute tasks:
- Each task requires a unique ID (uid) for hash-based worker assignment
- Tasks with the same uid always go to the same worker queue
- Useful for maintaining ordering guarantees per entity

### Circuit Breaker Pattern (httpx/hystrix)

The hystrix client wraps the base httpx client with:
- Configurable timeout, max concurrent requests, error threshold
- Fallback function support
- Optional StatsD metrics collection
- Automatic retry with backoff strategies

### Repository Pattern (sietch)

Three connector implementations sharing the same interface:
- **InMemoryConnector**: Thread-safe in-memory storage for testing
- **CockroachDBConnector**: SQL-based with reflection for column mapping using `db` struct tags
- **RedisConnector**: JSON serialization with TTL support

Key considerations:
- All connectors require a `getID func(*T) ID` function
- CockroachDB uses transactions for batch operations
- Redis uses pipelines for batch operations
- Query filtering only supported by InMemory and CockroachDB

## Testing Requirements

### Running External Dependencies

Some packages require external services for testing:

**sietch (Redis tests):**
```bash
docker run -d -p 6379:6379 redis:alpine
go test ./sietch -run TestRedis
```

**eventbus (NATS tests):**
```bash
docker run -d -p 4222:4222 nats:latest
go test ./eventbus -run TestNats
```

### Mock Generation

The repository uses mockery for generating mocks:
- `sietch/mock/Repository.go`: Mock repository implementation
- `eventbus/mock/Bus.go`: Mock event bus implementation

## Key Design Considerations

### Error Handling

- **sietch** defines domain-specific errors: `ErrItemNotFound`, `ErrItemAlreadyExists`, `ErrNoUpdateItem`, `ErrNoDeleteItem`, `ErrUnsupportedOperation`
- **httpx** uses `errors.Join` for multi-error accumulation during retries
- Always check for nil inputs and validate parameters

### Context Usage

All packages that perform I/O operations require `context.Context`:
- Repository operations (sietch)
- Event bus message receiving (eventbus)
- Worker pool cancellation (wp)

### Retry Strategies (httpx)

The httpx package supports pluggable retry strategies:
- `NoRetrier`: No backoff between retries
- Implement `Retrier` interface with `NextInterval(retryCount int) time.Duration` for custom strategies
- Retries trigger on network errors and 5xx status codes

### Struct Tags for Mapping (sietch)

CockroachDB connector uses reflection with `db` tags:
```go
type Account struct {
    ID      int64 `db:"id"`      // First db-tagged field is treated as primary key
    Balance int   `db:"balance"`
}
```

### Configuration Management (cfgmng)

Uses Viper with mapstructure tags:
```go
type Config struct {
    AppName string `mapstructure:"app_name"`
    Port    int    `mapstructure:"port"`
}
```

## Module Information

- Repository: `github.com/seb7887/gofw`
- Each package has module path: `github.com/seb7887/gofw/<package-name>`
- License: MIT
