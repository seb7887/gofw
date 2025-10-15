# Sietch Package Enhancement Brainstorming

## Table of Contents
1. [Current State Analysis](#current-state-analysis)
2. [Filter System Improvements](#filter-system-improvements)
3. [Aggregation Functions](#aggregation-functions)
4. [Transaction Support](#transaction-support)
5. [Additional Feature Proposals](#additional-feature-proposals)
6. [Implementation Roadmap](#implementation-roadmap)

---

## Current State Analysis

### Existing Architecture

**Repository Interface:**
- Generic interface with `Repository[T any, ID comparable]`
- Basic CRUD operations: Create, Get, Update, Delete
- Batch operations: BatchCreate, BatchUpdate, BatchDelete
- Query with simple Filter support

**Three Connector Implementations:**
1. **InMemoryConnector**: Thread-safe in-memory storage using map[ID]*T
2. **CockroachDBConnector**: PostgreSQL/CockroachDB using pgxpool
3. **RedisConnector**: Redis-based caching with TTL support

**Current Filter System:**
```go
type Condition struct {
    Field    string
    Operator string  // "=", "!=", ">", "<", ">=", "<="
    Value    any
}

type Filter struct {
    Conditions []Condition  // All conditions are ANDed together
}
```

### Identified Issues

#### Filter Problems

1. **Limited Operators**: Only basic comparison operators, no IN, LIKE, IS NULL, BETWEEN
2. **No Logical Operators**: Only AND logic, no OR or NOT support
3. **No Nested Conditions**: Can't build complex queries like `(A OR B) AND C`
4. **Field Name Inconsistency**:
   - InMemory: Uses reflection with field name capitalization (`strings.ToTitle`)
   - CockroachDB: Uses field names directly as column names
   - This creates confusion and potential bugs
5. **No Type Safety**: Operator is a string, easy to make typos
6. **No SQL Injection Protection for Filter Fields**: While table/column names are validated at connector creation, filter field names aren't validated
7. **Missing Query Features**:
   - No sorting (ORDER BY)
   - No pagination (LIMIT/OFFSET)
   - No result limiting
   - No DISTINCT
8. **Backend Inconsistency**: Redis doesn't support Query at all (returns ErrUnsupportedOperation)

#### Transaction Limitations

1. **No User-Facing Transaction API**: Internal transactions exist for batch operations but users can't create their own
2. **No Cross-Repository Transactions**: Can't perform operations across multiple repositories in a single transaction
3. **No Explicit Control**: No Begin/Commit/Rollback exposed to users
4. **Limited Atomic Operations**: Can only use batch operations for atomicity

#### Aggregation Absence

1. **No Aggregation Functions**: No COUNT, SUM, AVG, MIN, MAX
2. **No GROUP BY Support**: Can't aggregate by field
3. **No HAVING Clause**: Can't filter aggregated results

---

## Filter System Improvements

### Proposal 1: Enhanced Filter with Logical Operators

**Complexity**: Medium | **DX Impact**: High | **Backend Support**: CockroachDB, InMemory

```go
// New operator types
type LogicalOperator string

const (
    LogicalAND LogicalOperator = "AND"
    LogicalOR  LogicalOperator = "OR"
    LogicalNOT LogicalOperator = "NOT"
)

type ComparisonOperator string

const (
    OpEqual              ComparisonOperator = "="
    OpNotEqual           ComparisonOperator = "!="
    OpGreaterThan        ComparisonOperator = ">"
    OpLessThan           ComparisonOperator = "<"
    OpGreaterThanOrEqual ComparisonOperator = ">="
    OpLessThanOrEqual    ComparisonOperator = "<="
    OpIn                 ComparisonOperator = "IN"
    OpNotIn              ComparisonOperator = "NOT IN"
    OpLike               ComparisonOperator = "LIKE"
    OpILike              ComparisonOperator = "ILIKE"  // Case-insensitive
    OpIsNull             ComparisonOperator = "IS NULL"
    OpIsNotNull          ComparisonOperator = "IS NOT NULL"
    OpBetween            ComparisonOperator = "BETWEEN"
)

// Redesigned Condition with nesting support
type Condition struct {
    // For leaf conditions (actual comparisons)
    Field    string
    Operator ComparisonOperator
    Value    any

    // For composite conditions (logical grouping)
    LogicalOp  LogicalOperator
    Conditions []Condition  // Nested conditions
}

// Helper builder methods for better DX
type FilterBuilder struct {
    conditions []Condition
}

func NewFilter() *FilterBuilder {
    return &FilterBuilder{}
}

func (fb *FilterBuilder) Where(field string, op ComparisonOperator, value any) *FilterBuilder {
    fb.conditions = append(fb.conditions, Condition{
        Field:    field,
        Operator: op,
        Value:    value,
    })
    return fb
}

func (fb *FilterBuilder) And(conditions ...Condition) *FilterBuilder {
    fb.conditions = append(fb.conditions, Condition{
        LogicalOp:  LogicalAND,
        Conditions: conditions,
    })
    return fb
}

func (fb *FilterBuilder) Or(conditions ...Condition) *FilterBuilder {
    fb.conditions = append(fb.conditions, Condition{
        LogicalOp:  LogicalOR,
        Conditions: conditions,
    })
    return fb
}

func (fb *FilterBuilder) Build() *Filter {
    return &Filter{
        Conditions: fb.conditions,
        Sort:       fb.sort,
        Limit:      fb.limit,
        Offset:     fb.offset,
    }
}

// Usage example:
filter := NewFilter().
    Where("balance", OpGreaterThan, 1000).
    Or(
        Condition{Field: "status", Operator: OpEqual, Value: "premium"},
        Condition{Field: "age", Operator: OpGreaterThan, 18},
    ).
    Build()
// SQL: WHERE balance > 1000 OR (status = 'premium' AND age > 18)
```

**Pros:**
- Flexible and powerful query building
- Type-safe operators using constants
- Builder pattern improves developer experience
- Supports complex nested conditions

**Cons:**
- Breaking change to existing Filter structure
- More complex to implement for InMemory connector
- Redis still can't support this (would need secondary indexing)

**Implementation Notes:**
- CockroachDB: Recursively build SQL WHERE clauses
- InMemory: Recursively evaluate conditions with reflection
- Redis: Continue returning ErrUnsupportedOperation or implement with RediSearch module

---

### Proposal 2: Sorting, Pagination, and Limiting

**Complexity**: Low | **DX Impact**: High | **Backend Support**: CockroachDB, InMemory

```go
type SortDirection string

const (
    SortAsc  SortDirection = "ASC"
    SortDesc SortDirection = "DESC"
)

type SortField struct {
    Field     string
    Direction SortDirection
}

// Enhanced Filter with query modifiers
type Filter struct {
    Conditions []Condition
    Sort       []SortField  // Multiple fields for composite sorting
    Limit      *int         // Pointer to distinguish between 0 and not set
    Offset     *int
    Distinct   bool
}

// Builder methods
func (fb *FilterBuilder) OrderBy(field string, direction SortDirection) *FilterBuilder {
    fb.sort = append(fb.sort, SortField{Field: field, Direction: direction})
    return fb
}

func (fb *FilterBuilder) Limit(n int) *FilterBuilder {
    fb.limit = &n
    return fb
}

func (fb *FilterBuilder) Offset(n int) *FilterBuilder {
    fb.offset = &n
    return fb
}

func (fb *FilterBuilder) Distinct() *FilterBuilder {
    fb.distinct = true
    return fb
}

// Usage:
results, err := repo.Query(ctx, NewFilter().
    Where("balance", OpGreaterThan, 100).
    OrderBy("balance", SortDesc).
    Limit(10).
    Offset(20).
    Build())
```

**Pros:**
- Essential for real-world applications
- Simple to implement
- Non-breaking addition to Filter struct
- Greatly improves usability

**Cons:**
- InMemory sorting requires reflection or interface{} sorting
- Need to handle nil pointers carefully

---

### Proposal 3: Type-Safe Query Builder (Alternative Approach)

**Complexity**: High | **DX Impact**: Very High | **Backend Support**: All

```go
// Generic query builder with compile-time type safety
type Query[T any] struct {
    filters    []filterFunc[T]
    sorts      []sortFunc[T]
    limit      *int
    offset     *int
}

type filterFunc[T any] func(*T) bool
type sortFunc[T any] func(a, b *T) bool

func NewQuery[T any]() *Query[T] {
    return &Query[T]{}
}

func (q *Query[T]) Where(fn filterFunc[T]) *Query[T] {
    q.filters = append(q.filters, fn)
    return q
}

func (q *Query[T]) OrderBy(fn sortFunc[T]) *Query[T] {
    q.sorts = append(q.sorts, fn)
    return q
}

// Usage with type safety:
query := NewQuery[Account]().
    Where(func(a *Account) bool {
        return a.Balance > 1000 && a.Status == "active"
    }).
    OrderBy(func(a, b *Account) bool {
        return a.Balance > b.Balance
    }).
    Limit(10)

results, err := repo.QueryTyped(ctx, query)
```

**Pros:**
- Complete compile-time type safety
- No reflection needed at query time
- IDE autocomplete works perfectly
- Most intuitive for Go developers

**Cons:**
- Can't translate to SQL easily (needs to fetch all and filter in-memory)
- Performance issues for large datasets
- Doesn't leverage database indexing
- Breaking change requiring new interface method

**Verdict**: Good for InMemory, problematic for SQL databases. Could be offered as a separate method.

---

### Proposal 4: Field Name Standardization

**Complexity**: Low | **DX Impact**: Medium | **Backend Support**: All

**Problem**: InMemory capitalizes field names, CockroachDB doesn't, causing confusion.

**Solution**: Use struct tags consistently across all connectors.

```go
type Account struct {
    ID      int64  `db:"id" json:"id" repo:"id"`
    Balance int    `db:"balance" json:"balance" repo:"balance"`
    Status  string `db:"status" json:"status" repo:"status"`
}

// New tag parser utility
func getFieldMapping[T any]() (map[string]string, error) {
    var t T
    typ := reflect.TypeOf(t)
    if typ.Kind() == reflect.Ptr {
        typ = typ.Elem()
    }

    mapping := make(map[string]string)
    for i := 0; i < typ.NumField(); i++ {
        field := typ.Field(i)

        // Try multiple tags in order of preference
        tagName := field.Tag.Get("repo")
        if tagName == "" {
            tagName = field.Tag.Get("db")
        }
        if tagName == "" {
            tagName = field.Tag.Get("json")
        }
        if tagName == "" {
            tagName = strings.ToLower(field.Name)
        }

        mapping[tagName] = field.Name
    }

    return mapping, nil
}
```

**Pros:**
- Consistent behavior across all connectors
- Leverages existing struct tags
- Clear contract for users
- Backwards compatible with existing code

**Cons:**
- Slight performance overhead for InMemory (one-time map building)
- Requires documentation update

---

### Proposal 5: SQL Injection Protection for Filter Fields

**Complexity**: Low | **DX Impact**: Low (transparent) | **Backend Support**: CockroachDB

```go
// Add validation to query builder
func (r *CockroachDBConnector[T, ID]) validateFilterField(field string) error {
    // Check if field exists in known columns
    found := false
    for _, col := range r.columns {
        if col == field {
            found = true
            break
        }
    }

    if !found {
        return fmt.Errorf("unknown field '%s' for filtering", field)
    }

    // Additional validation
    return sanitizeIdentifier(field)
}

// Use in queryBuilder:
func (r *CockroachDBConnector[T, ID]) queryBuilder(filter *Filter) (string, []any, error) {
    // ... existing code ...

    for _, condition := range filter.Conditions {
        if err := r.validateFilterField(condition.Field); err != nil {
            return "", nil, err
        }
        // ... build query ...
    }
}
```

**Pros:**
- Prevents SQL injection through field names
- Catches typos early
- Minimal performance impact
- Easy to implement

**Cons:**
- Slightly more verbose error handling
- Need to update Repository interface signature (breaking change)

---

### Recommended Filter Implementation Strategy

**Phase 1: Non-Breaking Enhancements**
1. Add type-safe operator constants
2. Add Sort, Limit, Offset, Distinct to Filter struct
3. Implement field name validation
4. Add builder pattern as helpers (keep existing Filter struct compatible)

**Phase 2: Advanced Features**
5. Add IN, LIKE, IS NULL, BETWEEN operators
6. Implement logical operators (OR, NOT) with nested conditions
7. Add field mapping standardization

**Phase 3: Optional Type-Safe Alternative**
8. Add QueryTyped method with functional filter approach (non-breaking addition)

---

## Aggregation Functions

### Current Gap

No way to perform aggregations like COUNT, SUM, AVG, MIN, MAX, or GROUP BY operations.

### Proposal 1: Extend Repository Interface with Aggregate Methods

**Complexity**: Medium | **DX Impact**: High | **Backend Support**: CockroachDB, InMemory (partial)

```go
// New aggregate result type
type AggregateResult struct {
    Count  *int64
    Sum    *float64
    Avg    *float64
    Min    any
    Max    any
    Groups map[string][]AggregateResult  // For GROUP BY
}

// Aggregate operation specification
type AggregateOp string

const (
    AggCount AggregateOp = "COUNT"
    AggSum   AggregateOp = "SUM"
    AggAvg   AggregateOp = "AVG"
    AggMin   AggregateOp = "MIN"
    AggMax   AggregateOp = "MAX"
)

type AggregateQuery struct {
    Operations []struct {
        Op    AggregateOp
        Field string  // Empty for COUNT(*)
    }
    Filter  *Filter
    GroupBy []string
    Having  *Filter  // Filter on aggregated results
}

// Add to Repository interface:
type Repository[T any, ID comparable] interface {
    // ... existing methods ...

    Aggregate(ctx context.Context, query *AggregateQuery) (*AggregateResult, error)
    Count(ctx context.Context, filter *Filter) (int64, error)  // Convenience method
}

// Usage examples:
// Simple count
count, err := repo.Count(ctx, NewFilter().Where("status", OpEqual, "active").Build())

// Complex aggregation
result, err := repo.Aggregate(ctx, &AggregateQuery{
    Operations: []struct{Op AggregateOp; Field string}{
        {Op: AggCount, Field: ""},
        {Op: AggSum, Field: "balance"},
        {Op: AggAvg, Field: "balance"},
    },
    Filter: NewFilter().Where("status", OpEqual, "active").Build(),
    GroupBy: []string{"account_type"},
})

// Access results:
fmt.Printf("Total accounts: %d\n", *result.Count)
fmt.Printf("Total balance: %.2f\n", *result.Sum)
fmt.Printf("Average balance: %.2f\n", *result.Avg)

// Grouped results:
for accountType, groupResult := range result.Groups {
    fmt.Printf("%s: %d accounts, avg: %.2f\n",
        accountType, *groupResult[0].Count, *groupResult[0].Avg)
}
```

**Pros:**
- Clean interface extension
- Flexible and powerful
- Leverages database capabilities
- Intuitive usage

**Cons:**
- Breaking change to Repository interface (need to update all connectors)
- InMemory implementation complex for GROUP BY
- Redis can't support this without secondary indexing
- AggregateResult struct might be awkward with many nil pointers

---

### Proposal 2: Dedicated Aggregator Interface

**Complexity**: Medium | **DX Impact**: Medium | **Backend Support**: CockroachDB, InMemory

```go
// Separate interface for aggregation operations
type Aggregator[T any, ID comparable] interface {
    Count(ctx context.Context, filter *Filter) (int64, error)
    Sum(ctx context.Context, field string, filter *Filter) (float64, error)
    Avg(ctx context.Context, field string, filter *Filter) (float64, error)
    Min(ctx context.Context, field string, filter *Filter) (any, error)
    Max(ctx context.Context, field string, filter *Filter) (any, error)
    GroupBy(ctx context.Context, fields []string, aggregates []AggregateOp, filter *Filter) (map[string]*AggregateResult, error)
}

// Connectors optionally implement Aggregator
type CockroachDBConnector[T, ID] struct {
    // ... existing fields ...
}

// Type assertion to check if connector supports aggregation:
if agg, ok := repo.(Aggregator[Account, int64]); ok {
    count, err := agg.Count(ctx, filter)
}

// Or provide a helper:
func SupportsAggregation[T any, ID comparable](repo Repository[T, ID]) bool {
    _, ok := repo.(Aggregator[T, ID])
    return ok
}
```

**Pros:**
- Non-breaking change
- Clear separation of concerns
- Connectors can opt-in to aggregation support
- Simple per-operation methods

**Cons:**
- Requires type assertion
- GroupBy return type is complex
- No standardization across connectors

---

### Proposal 3: Functional Aggregation with Generics

**Complexity**: Low | **DX Impact**: High | **Backend Support**: InMemory only

```go
// Simple in-memory aggregation functions
func Count[T any, ID comparable](ctx context.Context, repo Repository[T, ID], filter *Filter) (int64, error) {
    results, err := repo.Query(ctx, filter)
    if err != nil {
        return 0, err
    }
    return int64(len(results)), nil
}

func Sum[T any, ID comparable](ctx context.Context, repo Repository[T, ID], filter *Filter, extractValue func(*T) float64) (float64, error) {
    results, err := repo.Query(ctx, filter)
    if err != nil {
        return 0, err
    }

    var sum float64
    for _, item := range results {
        sum += extractValue(&item)
    }
    return sum, nil
}

// Usage:
count, err := Count(ctx, repo, filter)
totalBalance, err := Sum(ctx, repo, filter, func(a *Account) float64 {
    return float64(a.Balance)
})

avgBalance := totalBalance / float64(count)
```

**Pros:**
- No interface changes
- Works with any Repository implementation
- Type-safe value extraction
- Simple to implement

**Cons:**
- Fetches all records (inefficient for large datasets)
- Doesn't leverage database aggregation
- No GROUP BY support
- Not suitable for production with large tables

**Verdict**: Good for testing or small datasets, not recommended for production.

---

### Proposal 4: SQL-Specific Aggregate Methods

**Complexity**: Low | **DX Impact**: Medium | **Backend Support**: CockroachDB only

```go
// Add SQL-specific methods to CockroachDB connector
func (r *CockroachDBConnector[T, ID]) CountSQL(ctx context.Context, whereClause string, args ...any) (int64, error) {
    query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s",
        quoteIdentifier(r.tableName), whereClause)

    var count int64
    err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
    return count, err
}

func (r *CockroachDBConnector[T, ID]) AggregateSQL(ctx context.Context, selectClause string, whereClause string, args ...any) (*AggregateResult, error) {
    // Raw SQL execution
}

// Usage:
count, err := cockroachRepo.CountSQL(ctx, "balance > $1", 1000)
result, err := cockroachRepo.AggregateSQL(ctx,
    "COUNT(*), SUM(balance), AVG(balance)",
    "status = $1",
    "active")
```

**Pros:**
- Maximum flexibility
- Leverages full SQL power
- No abstraction overhead

**Cons:**
- SQL-specific (not generic)
- Loses type safety
- Not consistent with Repository pattern
- SQL injection risk if not careful

**Verdict**: Could be useful as an "escape hatch" but shouldn't be the primary API.

---

### Recommended Aggregation Strategy

**Option A: Interface Extension (Recommended for Production)**
1. Extend Repository interface with Count(ctx, filter) method (most common use case)
2. Add optional Aggregator interface for advanced operations
3. CockroachDB and InMemory implement full Aggregator
4. Redis returns ErrUnsupportedOperation

**Option B: Separate Package (Lower Risk)**
1. Create `sietch/aggregate` package with Aggregator interface
2. Connectors implement aggregate.Aggregator if capable
3. No changes to core Repository interface
4. Users import aggregate package when needed

**Recommendation**: Start with Option A, just adding Count() to Repository interface. Add Aggregator interface in phase 2 if needed.

---

## Transaction Support

### Current Gap

- No user-facing transaction API
- Can't execute multiple operations atomically across repositories
- Can't manually control transaction boundaries
- Batch operations use internal transactions, but users can't create custom atomic operations

### Proposal 1: Context-Based Transaction Propagation

**Complexity**: Medium | **DX Impact**: High | **Backend Support**: CockroachDB

```go
// Transaction manager interface
type TxManager interface {
    BeginTx(ctx context.Context) (context.Context, error)
    CommitTx(ctx context.Context) error
    RollbackTx(ctx context.Context) error
}

// Add to CockroachDB connector
type CockroachDBConnector[T, ID] struct {
    pool      *pgxpool.Pool
    tableName string
    getID     func(*T) ID
    columns   []string
}

// Transaction key type for context
type txKey struct{}

// Transaction-aware operations
func (r *CockroachDBConnector[T, ID]) BeginTx(ctx context.Context) (context.Context, error) {
    tx, err := r.pool.Begin(ctx)
    if err != nil {
        return ctx, err
    }
    return context.WithValue(ctx, txKey{}, tx), nil
}

func (r *CockroachDBConnector[T, ID]) CommitTx(ctx context.Context) error {
    tx := ctx.Value(txKey{})
    if tx == nil {
        return fmt.Errorf("no transaction in context")
    }
    return tx.(pgx.Tx).Commit(ctx)
}

func (r *CockroachDBConnector[T, ID]) RollbackTx(ctx context.Context) error {
    tx := ctx.Value(txKey{})
    if tx == nil {
        return fmt.Errorf("no transaction in context")
    }
    return tx.(pgx.Tx).Rollback(ctx)
}

// Update operations to check context for transaction
func (r *CockroachDBConnector[T, ID]) getQueryable(ctx context.Context) Queryable {
    if tx := ctx.Value(txKey{}); tx != nil {
        return tx.(pgx.Tx)
    }
    return r.pool
}

// Queryable interface (sql.Tx and pgxpool.Pool both satisfy this)
type Queryable interface {
    Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
    Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
    QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Usage:
txCtx, err := accountRepo.BeginTx(ctx)
if err != nil {
    return err
}
defer accountRepo.RollbackTx(txCtx)  // Rollback if not committed

// All operations use txCtx
if err := accountRepo.Update(txCtx, &account); err != nil {
    return err
}

if err := auditRepo.Create(txCtx, &auditLog); err != nil {
    return err
}

// Commit transaction
if err := accountRepo.CommitTx(txCtx); err != nil {
    return err
}
```

**Pros:**
- Idiomatic Go pattern (context-based)
- Works across multiple repositories (they share the context)
- No new abstractions needed
- Automatic rollback with defer
- Non-breaking (adds new methods)

**Cons:**
- Requires type assertion internally
- Context value could be misused
- Need to ensure all repos use the same pool
- Defer rollback might hide errors

---

### Proposal 2: Explicit Transaction Interface

**Complexity**: High | **DX Impact**: High | **Backend Support**: CockroachDB

```go
// Transaction interface
type Tx[T any, ID comparable] interface {
    Repository[T, ID]  // Embeds all repository methods
    Commit() error
    Rollback() error
}

// Add to Repository interface
type Repository[T any, ID comparable] interface {
    // ... existing methods ...

    Begin(ctx context.Context) (Tx[T, ID], error)
}

// Implementation
type CockroachDBTx[T any, ID comparable] struct {
    *CockroachDBConnector[T, ID]  // Embed connector
    tx     pgx.Tx
    ctx    context.Context
}

func (r *CockroachDBConnector[T, ID]) Begin(ctx context.Context) (Tx[T, ID], error) {
    tx, err := r.pool.Begin(ctx)
    if err != nil {
        return nil, err
    }

    return &CockroachDBTx[T, ID]{
        CockroachDBConnector: r,
        tx:                   tx,
        ctx:                  ctx,
    }, nil
}

// Override methods to use tx instead of pool
func (t *CockroachDBTx[T, ID]) Create(ctx context.Context, item *T) error {
    values, err := t.getValues(item)
    if err != nil {
        return err
    }

    query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
        quoteIdentifier(t.tableName),
        joinQuotedColumns(t.columns),
        buildPlaceholders(len(t.columns)))

    _, err = t.tx.Exec(ctx, query, values...)
    return err
}

func (t *CockroachDBTx[T, ID]) Commit() error {
    return t.tx.Commit(t.ctx)
}

func (t *CockroachDBTx[T, ID]) Rollback() error {
    return t.tx.Rollback(t.ctx)
}

// Usage:
tx, err := accountRepo.Begin(ctx)
if err != nil {
    return err
}
defer tx.Rollback()  // Safe to call even after Commit

account.Balance -= 100
if err := tx.Update(ctx, &account); err != nil {
    return err  // Auto-rollback via defer
}

otherAccount.Balance += 100
if err := tx.Update(ctx, &otherAccount); err != nil {
    return err
}

if err := tx.Commit(); err != nil {
    return err
}
```

**Pros:**
- Clear, explicit API
- Type-safe transaction object
- Familiar to developers (similar to database/sql)
- Compiler enforces transaction usage
- Can't accidentally use wrong context

**Cons:**
- Breaking change (adds method to Repository interface)
- Can't share transaction across different repository instances
- More code duplication (need to implement Tx type for each connector)
- Complex generic types

---

### Proposal 3: Closure-Based Transaction Function (Recommended)

**Complexity**: Low | **DX Impact**: Very High | **Backend Support**: CockroachDB

```go
// Transaction function signature
type TxFunc[T any, ID comparable] func(repo Repository[T, ID]) error

// Add to Repository interface or as optional interface
type Transactional[T any, ID comparable] interface {
    WithTx(ctx context.Context, fn TxFunc[T, ID]) error
}

// Implementation
func (r *CockroachDBConnector[T, ID]) WithTx(ctx context.Context, fn TxFunc[T, ID]) error {
    tx, err := r.pool.Begin(ctx)
    if err != nil {
        return err
    }

    // Create a transaction-scoped repository
    txRepo := &CockroachDBConnector[T, ID]{
        pool:      &txWrapper{tx},  // Wrap tx to satisfy pool interface
        tableName: r.tableName,
        getID:     r.getID,
        columns:   r.columns,
    }

    defer func() {
        if p := recover(); p != nil {
            tx.Rollback(ctx)
            panic(p)
        }
    }()

    err = fn(txRepo)
    if err != nil {
        if rbErr := tx.Rollback(ctx); rbErr != nil {
            return fmt.Errorf("tx error: %v, rollback error: %v", err, rbErr)
        }
        return err
    }

    return tx.Commit(ctx)
}

// Usage:
err := accountRepo.WithTx(ctx, func(repo Repository[Account, int64]) error {
    // All operations within this function are transactional
    account.Balance -= 100
    if err := repo.Update(ctx, &account); err != nil {
        return err
    }

    otherAccount.Balance += 100
    if err := repo.Update(ctx, &otherAccount); err != nil {
        return err
    }

    // Return nil to commit, return error to rollback
    return nil
})
```

**Pros:**
- Clean, idiomatic Go API
- Automatic commit/rollback
- Panic-safe with defer
- Easy to use correctly
- No need for explicit Commit/Rollback calls
- Similar to patterns in ent, gorm

**Cons:**
- Limited to single repository
- Can't share transaction across multiple repositories
- Closure syntax might be unfamiliar to some developers
- Slightly less control over transaction lifecycle

---

### Proposal 4: Multi-Repository Transaction Manager

**Complexity**: High | **DX Impact**: High | **Backend Support**: CockroachDB

```go
// Global transaction manager
type TransactionManager struct {
    pool *pgxpool.Pool
}

func NewTransactionManager(pool *pgxpool.Pool) *TransactionManager {
    return &TransactionManager{pool: pool}
}

// Multi-repository transaction function
type MultiRepoTxFunc func(ctx context.Context) error

func (tm *TransactionManager) WithTx(ctx context.Context, fn MultiRepoTxFunc) error {
    tx, err := tm.pool.Begin(ctx)
    if err != nil {
        return err
    }

    // Inject transaction into context
    txCtx := context.WithValue(ctx, txKey{}, tx)

    defer func() {
        if p := recover(); p != nil {
            tx.Rollback(ctx)
            panic(p)
        }
    }()

    err = fn(txCtx)
    if err != nil {
        if rbErr := tx.Rollback(ctx); rbErr != nil {
            return fmt.Errorf("tx error: %v, rollback error: %v", err, rbErr)
        }
        return err
    }

    return tx.Commit(ctx)
}

// Repositories check context for active transaction
func (r *CockroachDBConnector[T, ID]) exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
    if tx := ctx.Value(txKey{}); tx != nil {
        return tx.(pgx.Tx).Exec(ctx, query, args...)
    }
    return r.pool.Exec(ctx, query, args...)
}

// Usage:
txManager := NewTransactionManager(pool)

err := txManager.WithTx(ctx, func(txCtx context.Context) error {
    // Multiple repositories can use txCtx
    if err := accountRepo.Update(txCtx, &account); err != nil {
        return err
    }

    if err := auditRepo.Create(txCtx, &auditLog); err != nil {
        return err
    }

    if err := notificationRepo.Create(txCtx, &notification); err != nil {
        return err
    }

    return nil
})
```

**Pros:**
- Works across multiple repositories
- Single transaction for entire operation
- Clean API with automatic commit/rollback
- Familiar pattern from other frameworks
- Panic-safe

**Cons:**
- Requires all repositories to share same pool
- Context injection pattern
- Need to ensure all repositories support transactions
- Global TransactionManager dependency

---

### Recommended Transaction Strategy

**Phase 1: Single-Repository Transactions**
- Implement Proposal 3 (Closure-Based) as `Transactional` interface
- CockroachDB implements it, others return ErrUnsupportedOperation
- Simple, safe, covers 80% of use cases

**Phase 2: Multi-Repository Transactions**
- Add TransactionManager (Proposal 4)
- Update all connectors to check context for active transaction
- Allows complex multi-repository transactions

**Alternative**: If multi-repository transactions are critical from day one, skip Phase 1 and go directly to Proposal 4.

---

## Additional Feature Proposals

### 1. Exists Method

**Complexity**: Low | **DX Impact**: Medium

```go
// Add to Repository interface
Exists(ctx context.Context, id ID) (bool, error)

// More efficient than Get for checking existence
// SQL: SELECT EXISTS(SELECT 1 FROM table WHERE id = $1)
```

### 2. Upsert (Insert or Update)

**Complexity**: Medium | **DX Impact**: High

```go
// Add to Repository interface
Upsert(ctx context.Context, item *T) error
BatchUpsert(ctx context.Context, items []T) error

// SQL: INSERT ... ON CONFLICT (id) DO UPDATE SET ...
// InMemory: Check existence, then Create or Update
```

### 3. Soft Delete Support

**Complexity**: Medium | **DX Impact**: High

```go
// Marker interface for soft delete support
type SoftDeletable interface {
    IsDeleted() bool
    SetDeleted(bool)
    GetDeletedAt() *time.Time
    SetDeletedAt(*time.Time)
}

// Connector option
func NewCockroachDBConnectorWithSoftDelete[T SoftDeletable, ID comparable](
    pool *pgxpool.Pool,
    tableName string,
    getID func(*T) ID,
) (*CockroachDBConnector[T, ID], error)

// Delete becomes soft delete if T implements SoftDeletable
// All queries automatically filter deleted records unless using QueryWithDeleted()
```

### 4. Hooks/Middleware System

**Complexity**: High | **DX Impact**: High

```go
// Hook interface
type Hook[T any] interface {
    BeforeCreate(ctx context.Context, item *T) error
    AfterCreate(ctx context.Context, item *T) error
    BeforeUpdate(ctx context.Context, item *T) error
    AfterUpdate(ctx context.Context, item *T) error
    BeforeDelete(ctx context.Context, id any) error
    AfterDelete(ctx context.Context, id any) error
}

// Add hooks to connector
func (r *CockroachDBConnector[T, ID]) AddHook(hook Hook[T]) {
    r.hooks = append(r.hooks, hook)
}

// Use cases:
// - Automatic timestamp updates
// - Audit logging
// - Cache invalidation
// - Event publishing
// - Validation
```

### 5. Query Builder with Method Chaining

**Complexity**: High | **DX Impact**: Very High

Already partially covered in Filter proposals, but could be extended:

```go
results, err := repo.Query(ctx).
    Where("balance", OpGreaterThan, 1000).
    Where("status", OpEqual, "active").
    OrderBy("created_at", SortDesc).
    Limit(10).
    Execute()

// Or with Begin for query reuse:
query := repo.Query(ctx).
    Where("status", OpEqual, "active")

active, err := query.Count()
avgBalance, err := query.Avg("balance")
```

### 6. Relationship Loading (Joins)

**Complexity**: Very High | **DX Impact**: High | **Backend Support**: CockroachDB only

```go
type Account struct {
    ID      int64    `db:"id"`
    Balance int      `db:"balance"`
    UserID  int64    `db:"user_id"`
    User    *User    `db:"-" relation:"user_id"`  // Not stored, loaded via join
}

// Relationship loader
func (r *CockroachDBConnector[T, ID]) QueryWithRelations(
    ctx context.Context,
    filter *Filter,
    relations ...string,
) ([]T, error)

// Usage:
accounts, err := accountRepo.QueryWithRelations(ctx, filter, "User")
// SQL: SELECT accounts.*, users.* FROM accounts JOIN users ON accounts.user_id = users.id
```

**Note**: This is very complex and might be better suited for an ORM. Could be out of scope for sietch.

### 7. Bulk Operations with Result Feedback

**Complexity**: Low | **DX Impact**: Medium

```go
type BatchResult struct {
    SuccessCount int
    FailureCount int
    Errors       map[int]error  // Index -> error mapping
}

// Enhanced batch operations
BatchCreateWithResults(ctx context.Context, items []T) (*BatchResult, error)
BatchUpdateWithResults(ctx context.Context, items []T) (*BatchResult, error)
BatchDeleteWithResults(ctx context.Context, ids []ID) (*BatchResult, error)
```

### 8. Pagination Helper

**Complexity**: Low | **DX Impact**: High

```go
type Page[T any] struct {
    Items      []T
    Total      int64
    Page       int
    PageSize   int
    TotalPages int
    HasNext    bool
    HasPrev    bool
}

// Convenience method
func Paginate[T any, ID comparable](
    ctx context.Context,
    repo Repository[T, ID],
    filter *Filter,
    page int,
    pageSize int,
) (*Page[T], error) {
    // Calculate offset
    offset := (page - 1) * pageSize

    // Get total count
    totalFilter := filter.Clone()
    totalFilter.Limit = nil
    totalFilter.Offset = nil
    total, err := repo.Count(ctx, totalFilter)

    // Get page items
    filter.Limit = &pageSize
    filter.Offset = &offset
    items, err := repo.Query(ctx, filter)

    return &Page[T]{
        Items:      items,
        Total:      total,
        Page:       page,
        PageSize:   pageSize,
        TotalPages: int(math.Ceil(float64(total) / float64(pageSize))),
        HasNext:    page < totalPages,
        HasPrev:    page > 1,
    }
}
```

### 9. Caching Layer

**Complexity**: Medium | **DX Impact**: High

```go
// Cache-aware repository wrapper
type CachedRepository[T any, ID comparable] struct {
    base  Repository[T, ID]
    cache Repository[T, ID]  // Redis connector
    ttl   time.Duration
}

func NewCachedRepository[T any, ID comparable](
    base Repository[T, ID],
    cache Repository[T, ID],
    ttl time.Duration,
) *CachedRepository[T, ID]

// Get checks cache first, then base
func (r *CachedRepository[T, ID]) Get(ctx context.Context, id ID) (*T, error) {
    // Try cache
    item, err := r.cache.Get(ctx, id)
    if err == nil {
        return item, nil
    }

    // Cache miss, get from base
    item, err = r.base.Get(ctx, id)
    if err != nil {
        return nil, err
    }

    // Update cache (async)
    go r.cache.Create(context.Background(), item)

    return item, nil
}

// Usage:
cachedRepo := NewCachedRepository(cockroachRepo, redisRepo, 5*time.Minute)
account, err := cachedRepo.Get(ctx, id)  // Automatically cached
```

### 10. Event Sourcing Support

**Complexity**: Very High | **DX Impact**: Variable

```go
// Event interface
type Event interface {
    GetAggregateID() any
    GetEventType() string
    GetTimestamp() time.Time
}

// Event store repository
type EventStore[E Event] interface {
    AppendEvent(ctx context.Context, event E) error
    GetEvents(ctx context.Context, aggregateID any) ([]E, error)
    GetEventsByType(ctx context.Context, eventType string) ([]E, error)
}
```

**Note**: Event sourcing is a large architectural pattern. Might be better as a separate package.

### 11. Schema Migration Helpers

**Complexity**: Medium | **DX Impact**: High

```go
// Schema definition from struct
func (r *CockroachDBConnector[T, ID]) CreateTable(ctx context.Context) error
func (r *CockroachDBConnector[T, ID]) DropTable(ctx context.Context) error
func (r *CockroachDBConnector[T, ID]) CreateIndexes(ctx context.Context, indexes ...Index) error

// Useful for testing and development
```

### 12. Query Logging and Metrics

**Complexity**: Low | **DX Impact**: Medium

```go
// Logger interface
type QueryLogger interface {
    LogQuery(ctx context.Context, query string, args []any, duration time.Duration, err error)
}

// Add to connector
func (r *CockroachDBConnector[T, ID]) SetLogger(logger QueryLogger) {
    r.logger = logger
}

// Automatically log all queries with timing
```

---

## Implementation Roadmap

### Priority 1: Essential Features (MVP)

1. **Filter Improvements**
   - Type-safe operator constants ✅ Low complexity, high value
   - Sort, Limit, Offset support ✅ Essential for production
   - Field name validation ✅ Security improvement
   - IN, LIKE, IS NULL operators ✅ Common use cases

2. **Aggregations**
   - Count() method ✅ Most commonly needed
   - Optional Aggregator interface for Sum/Avg/Min/Max

3. **Transactions**
   - Closure-based transaction (Proposal 3) ✅ Clean API, 80% use case coverage

**Timeline**: 2-3 weeks
**Breaking Changes**: Repository interface gets Count() and optional Transactional interface

---

### Priority 2: Enhanced Features

4. **Advanced Filters**
   - OR/NOT logical operators
   - Nested conditions
   - Builder pattern improvements

5. **Transactions**
   - Multi-repository transaction manager (Proposal 4)
   - Context-based transaction propagation

6. **Utility Methods**
   - Exists() method
   - Upsert() and BatchUpsert()
   - Pagination helper

**Timeline**: 2-3 weeks
**Breaking Changes**: Filter struct changes (may need versioning)

---

### Priority 3: Advanced Features

7. **Soft Delete Support**
8. **Hooks/Middleware System**
9. **Caching Layer**
10. **Query Logging**
11. **Schema Migration Helpers**

**Timeline**: 4-6 weeks
**Breaking Changes**: Minimal, mostly additive

---

### Priority 4: Future Considerations

12. **Relationship Loading** (Complex, might need separate ORM-like package)
13. **Event Sourcing** (Separate package recommended)
14. **Type-Safe Query Builder** (Alternative API, could be separate package)

**Timeline**: TBD
**Breaking Changes**: None (separate packages)

---

## Implementation Best Practices

### Backwards Compatibility

1. **Add, Don't Change**: Prefer adding new methods over changing existing ones
2. **Versioning**: Consider `v2` package if breaking changes are necessary
3. **Deprecation**: Mark old APIs as deprecated with clear migration path
4. **Feature Flags**: Use build tags or options for experimental features

### Testing Strategy

1. **Unit Tests**: Test query builders, filter logic, validation
2. **Integration Tests**: Test against real CockroachDB and Redis
3. **Benchmark Tests**: Ensure performance doesn't degrade
4. **Compatibility Tests**: Test all connectors implement interface correctly

### Documentation

1. **Godoc**: Comprehensive package and method documentation
2. **Examples**: Real-world usage examples for each feature
3. **Migration Guide**: Clear upgrade path for breaking changes
4. **Performance Notes**: Document performance characteristics of each operation

### Error Handling

1. **Sentinel Errors**: Use predefined errors (like ErrItemNotFound)
2. **Error Wrapping**: Use fmt.Errorf with %w for error chains
3. **Validation Errors**: Clear error messages for invalid inputs
4. **Transaction Errors**: Distinguish between transaction and operation errors

---

## Conclusion

This brainstorming document outlines a comprehensive roadmap for enhancing the sietch package. The proposals are designed to be:

- **Scalable**: Handle large datasets and complex queries efficiently
- **Maintainable**: Clean interfaces, well-tested, and documented
- **Developer-Friendly**: Intuitive APIs, type-safe, and great IDE support

The phased approach allows for incremental adoption without forcing users to upgrade immediately. Each phase delivers tangible value while maintaining backwards compatibility where possible.

### Next Steps

1. **Review & Feedback**: Gather team feedback on priorities and approaches
2. **Proof of Concept**: Implement Priority 1 features in a feature branch
3. **Performance Testing**: Benchmark proposed changes against current implementation
4. **API Design Review**: Finalize interfaces before implementation
5. **Implementation**: Execute roadmap in priority order

---

*Document Version*: 1.0
*Last Updated*: 2025-10-14
*Author*: Claude Code Analysis
