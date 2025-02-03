# Sietch

**Sietch** is a Go package that provides a unified, generic repository interface for performing CRUD operations across multiple database backends. Using Go generics and reflection, Sietch enables you to write data-access code in a backend-agnostic manner. Out-of-the-box implementations include:

- **InMemoryRepository**: Useful for testing and business logic prototyping.
- **CockroachRepository**: A generic implementation for CockroachDB using [pgxpool](https://github.com/jackc/pgx) that supports real CRUD operations.
- **RedisRepository**: A cache repository that serializes entities to JSON and supports setting a default TTL (time-to-live).

## Features

- **Unified CRUD Interface**: Define operations like `Create`, `Get`, `BatchCreate`, `Query`, `Update`, `BatchUpdate`, `Delete`, and `BatchDelete` in a single interface.
- **Backend Agnostic**: Write your business logic once and use dependency injection to switch between in-memory, SQL, or cache backends.
- **Generics and Reflection**: Automatically map struct fields (using `db` tags) to database columns, build SQL queries dynamically, and serialize/deserialize JSON for Redis.
- **Batch Operations**: Efficiently perform batch updates and deletes using transactions (for CockroachDB) or pipelines (for Redis).
- **Cache with TTL**: The Redis repository is designed for caching with a configurable default TTL.

## Requirements

- **Go 1.18+** (for generics support)
- For CockroachRepository: [pgxpool](https://github.com/jackc/pgx) v5
- For RedisRepository: [go-redis/redis/v8](https://github.com/go-redis/redis) v8

## Installation

Use `go get` to add Sietch to your module:

```sh
go get github.com/seb7887/gofw/sietch
