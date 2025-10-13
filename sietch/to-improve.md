# Sietch Module Improvements and Fixes

This document contains detailed improvements and fixes for the sietch module. Each item is written as a self-contained prompt that can be used by an AI agent without requiring additional context.

---

## 1. CRITICAL BUGS

### BUG-001: InMemoryConnector.Create returns wrong error when item already exists

**Priority:** CRITICAL
**File:** `inmemory.go:24-35`

**Problem:**
The `Create` method in `InMemoryConnector` returns `ErrItemNotFound` when an item already exists. This is incorrect - it should return `ErrItemAlreadyExists`.

**Current code (line 29-30):**
```go
if _, exists := r.data[id]; exists {
    return ErrItemNotFound  // WRONG ERROR
}
```

**Fix:**
Replace `ErrItemNotFound` with `ErrItemAlreadyExists` on line 30.

**Expected behavior:**
```go
if _, exists := r.data[id]; exists {
    return ErrItemAlreadyExists
}
```

**Acceptance criteria:**
- [ ] The Create method returns `ErrItemAlreadyExists` when attempting to create a duplicate item
- [ ] Existing test in `inmemory_test.go:22` passes with correct error validation
- [ ] Update test assertion to check for `ErrItemAlreadyExists` instead of generic error

**Test to update:**
In `inmemory_test.go`, test "create duplicated account" should verify:
```go
if err != ErrItemAlreadyExists {
    t.Errorf("expected ErrItemAlreadyExists, got: %v", err)
}
```

---

## 2. MISSING VALIDATIONS

### VAL-001: Add nil item validation in CockroachDBConnector.Create

**Priority:** HIGH
**File:** `cockroach.go:159-172`

**Problem:**
The `Create` method doesn't validate if the input `item` parameter is nil before calling `getValues(item)`. This can cause a panic if a nil pointer is passed.

**Current code (line 159-163):**
```go
func (r *CockroachDBConnector[T, ID]) Create(ctx context.Context, item *T) error {
    values, err := r.getValues(item)
    if err != nil {
        return err
    }
    // ...
}
```

**Fix:**
Add nil validation at the start of the method:

```go
func (r *CockroachDBConnector[T, ID]) Create(ctx context.Context, item *T) error {
    if item == nil {
        return errors.New("item cannot be nil")
    }
    values, err := r.getValues(item)
    if err != nil {
        return err
    }
    // ...
}
```

**Acceptance criteria:**
- [ ] Create method returns appropriate error when item is nil
- [ ] Add test case: `TestCockroachDBConnector_CreateNilValidation`
- [ ] Test verifies error message matches "item cannot be nil"

**Test to add:**
```go
func TestCockroachDBConnector_CreateNilValidation(t *testing.T) {
    conn := createTestConnector(t)
    ctx := context.Background()

    err := conn.Create(ctx, nil)
    if err == nil || err.Error() != "item cannot be nil" {
        t.Errorf("expected 'item cannot be nil' error, got: %v", err)
    }
}
```

---

### VAL-002: Add nil item validation in CockroachDBConnector.Update

**Priority:** HIGH
**File:** `cockroach.go:258-289`

**Problem:**
The `Update` method doesn't validate if the input `item` parameter is nil before calling `getValues(item)` and `getID(item)`. This can cause a panic.

**Current code (line 258-262):**
```go
func (r *CockroachDBConnector[T, ID]) Update(ctx context.Context, item *T) error {
    values, err := r.getValues(item)
    if err != nil {
        return err
    }
    // ...
}
```

**Fix:**
Add nil validation at the start of the method:

```go
func (r *CockroachDBConnector[T, ID]) Update(ctx context.Context, item *T) error {
    if item == nil {
        return errors.New("item cannot be nil")
    }
    values, err := r.getValues(item)
    if err != nil {
        return err
    }
    // ...
}
```

**Acceptance criteria:**
- [ ] Update method returns appropriate error when item is nil
- [ ] Add test case: `TestCockroachDBConnector_UpdateNilValidation`
- [ ] Test verifies error message matches "item cannot be nil"

---

### VAL-003: Add nil item validation in InMemoryConnector.Create and Update

**Priority:** HIGH
**Files:** `inmemory.go:24` and `inmemory.go:72`

**Problem:**
Both `Create` and `Update` methods don't validate nil items, which could cause panics when calling `getID(item)`.

**Fix for Create (add at line 25):**
```go
func (r *InMemoryConnector[T, ID]) Create(_ context.Context, item *T) error {
    if item == nil {
        return errors.New("item cannot be nil")
    }
    r.mu.Lock()
    defer r.mu.Unlock()
    // ...
}
```

**Fix for Update (add at line 73):**
```go
func (r *InMemoryConnector[T, ID]) Update(_ context.Context, item *T) error {
    if item == nil {
        return errors.New("item cannot be nil")
    }
    r.mu.Lock()
    defer r.mu.Unlock()
    // ...
}
```

**Acceptance criteria:**
- [ ] Both methods return appropriate error when item is nil
- [ ] Add test cases for both methods
- [ ] Error message is consistent: "item cannot be nil"

---

### VAL-004: Validate successful deletion in RedisConnector.Delete

**Priority:** MEDIUM
**File:** `redis.go:102-105`

**Problem:**
The `Delete` method doesn't verify if the key actually existed and was deleted. It should return `ErrItemNotFound` if the key didn't exist, similar to how other connectors handle deletion.

**Current code:**
```go
func (r *RedisConnector[T, ID]) Delete(ctx context.Context, id ID) error {
    key := r.keyFunc(id)
    return r.client.Del(ctx, key).Err()
}
```

**Fix:**
Check the result of Del command:

```go
func (r *RedisConnector[T, ID]) Delete(ctx context.Context, id ID) error {
    key := r.keyFunc(id)
    result, err := r.client.Del(ctx, key).Result()
    if err != nil {
        return err
    }
    if result == 0 {
        return ErrItemNotFound
    }
    return nil
}
```

**Acceptance criteria:**
- [ ] Delete returns `ErrItemNotFound` when key doesn't exist
- [ ] Delete returns nil when key was successfully deleted
- [ ] Add test: `TestRedisConnector_DeleteNonExisting`

**Test to add:**
```go
func TestRedisConnector_DeleteNonExisting(t *testing.T) {
    client, repo := setupRedisTest(t)
    defer client.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    // Try to delete non-existing item
    err := repo.Delete(ctx, 9999)
    if err != ErrItemNotFound {
        t.Errorf("expected ErrItemNotFound, got: %v", err)
    }
}
```

---

### VAL-005: Handle primary key constraint violations in CockroachDB

**Priority:** MEDIUM
**File:** `cockroach.go:159-172`

**Problem:**
When a duplicate key is inserted in CockroachDB, the database returns a constraint violation error, but this is not wrapped or converted to `ErrItemAlreadyExists` for consistency with other connectors.

**Fix:**
Wrap the error checking logic:

```go
func (r *CockroachDBConnector[T, ID]) Create(ctx context.Context, item *T) error {
    if item == nil {
        return errors.New("item cannot be nil")
    }

    values, err := r.getValues(item)
    if err != nil {
        return err
    }

    query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
        quoteIdentifier(r.tableName),
        joinQuotedColumns(r.columns),
        buildPlaceholders(len(r.columns)),
    )
    _, err = r.pool.Exec(ctx, query, values...)

    // Check for duplicate key error
    if err != nil && strings.Contains(err.Error(), "duplicate key") {
        return ErrItemAlreadyExists
    }

    return err
}
```

**Acceptance criteria:**
- [ ] Create returns `ErrItemAlreadyExists` on duplicate key violations
- [ ] Other database errors are returned as-is
- [ ] Add integration test with actual database connection

---

## 3. INCONSISTENCIES

### INC-001: Inconsistent empty slice handling across connectors

**Priority:** MEDIUM
**Files:** `inmemory.go:49-56`, `cockroach.go:192-229`, `redis.go:52-82`

**Problem:**
Different connectors handle empty slices inconsistently in batch operations:
- **InMemoryConnector**: Iterates over empty slice (no-op but still acquires locks)
- **RedisConnector**: Early returns for empty/nil slices (line 53-55)
- **CockroachDBConnector**: No validation, starts transaction for empty slice

**Fix:**
Standardize behavior across all connectors to early return for empty slices:

**InMemoryConnector.BatchCreate (add at line 50):**
```go
func (r *InMemoryConnector[T, ID]) BatchCreate(ctx context.Context, items []T) error {
    if len(items) == 0 {
        return nil
    }
    for _, item := range items {
        if err := r.Create(ctx, &item); err != nil {
            return err
        }
    }
    return nil
}
```

**Apply same pattern to:**
- `InMemoryConnector.BatchUpdate`
- `InMemoryConnector.BatchDelete`
- `CockroachDBConnector.BatchCreate`
- `CockroachDBConnector.BatchUpdate`
- `CockroachDBConnector.BatchDelete`

**Acceptance criteria:**
- [ ] All batch methods return early for empty slices
- [ ] No transactions or locks acquired for empty operations
- [ ] Consistent behavior across all three connectors
- [ ] Add tests verifying no side effects for empty batches

---

### INC-002: Inefficient locking in InMemoryConnector batch operations

**Priority:** MEDIUM
**Files:** `inmemory.go:49-56`, `85-92`, `106-113`

**Problem:**
Batch operations in InMemoryConnector acquire and release locks multiple times by calling individual Create/Update/Delete methods in a loop. This is inefficient and could cause inconsistent state if one operation fails mid-batch.

**Current approach:**
```go
func (r *InMemoryConnector[T, ID]) BatchCreate(ctx context.Context, items []T) error {
    for _, item := range items {
        if err := r.Create(ctx, &item); err != nil {  // Locks and unlocks each time
            return err
        }
    }
    return nil
}
```

**Fix:**
Acquire lock once for entire batch:

```go
func (r *InMemoryConnector[T, ID]) BatchCreate(ctx context.Context, items []T) error {
    if len(items) == 0 {
        return nil
    }

    r.mu.Lock()
    defer r.mu.Unlock()

    for _, item := range items {
        id := r.getID(&item)
        if _, exists := r.data[id]; exists {
            return ErrItemAlreadyExists
        }
        r.data[id] = &item
    }
    return nil
}
```

**Apply same pattern to BatchUpdate and BatchDelete.**

**Acceptance criteria:**
- [ ] Lock acquired only once per batch operation
- [ ] Atomic batch operations (all succeed or all fail)
- [ ] Significantly better performance for large batches
- [ ] Add benchmark comparing old vs new implementation

---

## 4. FUNCTIONALITY IMPROVEMENTS

### FEAT-001: Add support for OR operator in filter conditions

**Priority:** MEDIUM
**Files:** `filters.go`, `inmemory.go:115-164`, `cockroach.go:408-438`

**Problem:**
The current filtering system only supports AND logic between conditions. Many real-world queries need OR logic (e.g., "balance > 1000 OR id < 10").

**Current structure:**
```go
type Filter struct {
    Conditions []Condition
}
```

**Proposed solution:**
Add support for logical operators:

```go
type LogicalOperator string

const (
    LogicalAnd LogicalOperator = "AND"
    LogicalOr  LogicalOperator = "OR"
)

type Filter struct {
    Conditions []Condition
    Operator   LogicalOperator // Default to AND for backwards compatibility
}
```

**Update matchesCondition in inmemory.go:**
```go
func matchesCondition(item any, filter *Filter) bool {
    v := reflect.ValueOf(item)
    if v.Kind() == reflect.Ptr {
        v = v.Elem()
    }
    if v.Kind() != reflect.Struct {
        return false
    }

    // Default to AND if not specified
    operator := filter.Operator
    if operator == "" {
        operator = LogicalAnd
    }

    for _, condition := range filter.Conditions {
        fieldVal := v.FieldByName(strings.ToTitle(string(condition.Field[0])) + condition.Field[1:])
        if !fieldVal.IsValid() {
            return false
        }

        valueInterface := fieldVal.Interface()
        matches := evaluateCondition(valueInterface, condition)

        if operator == LogicalOr && matches {
            return true // Short-circuit for OR
        }
        if operator == LogicalAnd && !matches {
            return false // Short-circuit for AND
        }
    }

    // If we reach here with OR, nothing matched
    // If we reach here with AND, everything matched
    return operator == LogicalAnd
}

func evaluateCondition(fieldValue any, condition Condition) bool {
    switch condition.Operator {
    case "=":
        return reflect.DeepEqual(fieldValue, condition.Value)
    case "!=":
        return !reflect.DeepEqual(fieldValue, condition.Value)
    case ">":
        return compare(fieldValue, condition.Value) > 0
    case "<":
        return compare(fieldValue, condition.Value) < 0
    case ">=":
        return compare(fieldValue, condition.Value) >= 0
    case "<=":
        return compare(fieldValue, condition.Value) <= 0
    default:
        return false
    }
}
```

**Update queryBuilder in cockroach.go:**
```go
func (r *CockroachDBConnector[T, ID]) queryBuilder(filter *Filter) (string, []any) {
    whereClause := ""
    var args []any

    if filter == nil || len(filter.Conditions) == 0 {
        query := fmt.Sprintf("SELECT %s FROM %s",
            joinQuotedColumns(r.columns),
            quoteIdentifier(r.tableName),
        )
        return query, args
    }

    // Default to AND
    operator := string(filter.Operator)
    if operator == "" {
        operator = "AND"
    }

    for i, condition := range filter.Conditions {
        if i == 0 {
            whereClause = "WHERE "
        } else {
            whereClause += fmt.Sprintf(" %s ", operator)
        }
        whereClause += fmt.Sprintf("%s %s $%d", quoteIdentifier(condition.Field), condition.Operator, i+1)
        args = append(args, condition.Value)
    }

    query := fmt.Sprintf("SELECT %s FROM %s %s",
        joinQuotedColumns(r.columns),
        quoteIdentifier(r.tableName),
        whereClause,
    )

    return query, args
}
```

**Acceptance criteria:**
- [ ] Filter struct has Operator field with default AND behavior
- [ ] InMemoryConnector supports both AND and OR operators
- [ ] CockroachDBConnector generates correct SQL with OR
- [ ] Backward compatibility maintained (default to AND)
- [ ] Add tests for OR logic in both connectors
- [ ] Update documentation

---

### FEAT-002: Validate SQL operators in queryBuilder

**Priority:** HIGH
**File:** `cockroach.go:408-438`

**Problem:**
The `queryBuilder` method doesn't validate the operator in conditions. Malicious or invalid operators could be injected into SQL queries, potentially causing SQL injection vulnerabilities.

**Current code (line 427):**
```go
whereClause += fmt.Sprintf("%s %s $%d", quoteIdentifier(condition.Field), condition.Operator, i+1)
```

**Fix:**
Add operator validation:

```go
var allowedOperators = map[string]bool{
    "=":  true,
    "!=": true,
    ">":  true,
    "<":  true,
    ">=": true,
    "<=": true,
    "LIKE": true,
    "ILIKE": true,
    "IN": true,
    "NOT IN": true,
}

func isValidOperator(op string) bool {
    return allowedOperators[op]
}

func (r *CockroachDBConnector[T, ID]) queryBuilder(filter *Filter) (string, []any) {
    whereClause := ""
    var args []any

    if filter == nil || len(filter.Conditions) == 0 {
        query := fmt.Sprintf("SELECT %s FROM %s",
            joinQuotedColumns(r.columns),
            quoteIdentifier(r.tableName),
        )
        return query, args
    }

    // Validate all operators first
    for _, condition := range filter.Conditions {
        if !isValidOperator(condition.Operator) {
            // Return empty query and error indicators
            return "", nil
        }
    }

    for i, condition := range filter.Conditions {
        if i == 0 {
            whereClause = "WHERE "
        } else {
            whereClause += " AND "
        }
        whereClause += fmt.Sprintf("%s %s $%d", quoteIdentifier(condition.Field), condition.Operator, i+1)
        args = append(args, condition.Value)
    }

    query := fmt.Sprintf("SELECT %s FROM %s %s",
        joinQuotedColumns(r.columns),
        quoteIdentifier(r.tableName),
        whereClause,
    )

    return query, args
}
```

**Better approach - return error:**
Change `queryBuilder` signature to return error:

```go
func (r *CockroachDBConnector[T, ID]) queryBuilder(filter *Filter) (string, []any, error) {
    // ... validation ...
    if !isValidOperator(condition.Operator) {
        return "", nil, fmt.Errorf("invalid operator: %s", condition.Operator)
    }
    // ... rest of the code ...
    return query, args, nil
}
```

Then update `Query` method to handle the error.

**Acceptance criteria:**
- [ ] Only whitelisted operators are allowed
- [ ] Invalid operators return clear error
- [ ] Add test with invalid operator
- [ ] Update Query method signature and callers

---

### FEAT-003: Support partial updates in RedisConnector with field-level operations

**Priority:** LOW
**File:** `redis.go:88-93`

**Problem:**
The `Update` method in RedisConnector completely replaces the item by calling `Create`. There's no way to update specific fields without fetching, modifying, and writing back the entire object.

**Current implementation:**
```go
func (r *RedisConnector[T, ID]) Update(ctx context.Context, item *T) error {
    if item == nil {
        return errors.New("item cannot be nil")
    }
    return r.Create(ctx, item)
}
```

**Proposed solution:**
Add a new method for field-level updates using Redis HSET:

```go
// UpdateFields updates specific fields of an item in Redis
// fields is a map of field names to values
func (r *RedisConnector[T, ID]) UpdateFields(ctx context.Context, id ID, fields map[string]interface{}) error {
    if len(fields) == 0 {
        return nil
    }

    key := r.keyFunc(id)

    // Check if key exists
    exists, err := r.client.Exists(ctx, key).Result()
    if err != nil {
        return err
    }
    if exists == 0 {
        return ErrItemNotFound
    }

    // Get current item
    current, err := r.Get(ctx, id)
    if err != nil {
        return err
    }

    // Use reflection to update fields
    v := reflect.ValueOf(current).Elem()
    for fieldName, value := range fields {
        field := v.FieldByName(fieldName)
        if !field.IsValid() || !field.CanSet() {
            return fmt.Errorf("field %s is not valid or cannot be set", fieldName)
        }
        field.Set(reflect.ValueOf(value))
    }

    // Save updated item
    return r.Create(ctx, current)
}
```

**Alternative: Use Redis Hash structure:**
This requires restructuring how data is stored (use HSET/HGET instead of SET/GET with JSON).

**Acceptance criteria:**
- [ ] Add UpdateFields method to interface (optional)
- [ ] Method validates field existence and types
- [ ] Method returns ErrItemNotFound if item doesn't exist
- [ ] Add comprehensive tests
- [ ] Document that this is Redis-specific functionality

**Note:** This is a breaking change if added to the Repository interface. Consider creating a separate interface like `PartialUpdater[T, ID]`.

---

### FEAT-004: Add context cancellation checks in long-running loops

**Priority:** MEDIUM
**Files:** `inmemory.go` (batch methods), `cockroach.go` (batch methods)

**Problem:**
Batch operations don't check for context cancellation, which means they'll continue processing even if the context is cancelled. This can waste resources and delay shutdown.

**Fix for InMemoryConnector.BatchCreate:**
```go
func (r *InMemoryConnector[T, ID]) BatchCreate(ctx context.Context, items []T) error {
    if len(items) == 0 {
        return nil
    }

    r.mu.Lock()
    defer r.mu.Unlock()

    for i, item := range items {
        // Check context every N items (e.g., every 100)
        if i % 100 == 0 {
            select {
            case <-ctx.Done():
                return ctx.Err()
            default:
            }
        }

        id := r.getID(&item)
        if _, exists := r.data[id]; exists {
            return ErrItemAlreadyExists
        }
        r.data[id] = &item
    }
    return nil
}
```

**Fix for CockroachDBConnector batch methods:**
Add context check inside transaction loops (lines 217-226, 328-344, 395-403).

**Acceptance criteria:**
- [ ] All batch operations check context periodically
- [ ] Add test that cancels context mid-batch
- [ ] Verify proper cleanup on cancellation
- [ ] Balance between performance and responsiveness (check every ~100 items)

---

### FEAT-005: Add index support for InMemoryConnector queries

**Priority:** LOW
**File:** `inmemory.go:58-70`

**Problem:**
The `Query` method iterates over all items in memory for every query, which is O(n). For large datasets and frequent queries on indexed fields (like commonly queried fields), this is inefficient.

**Proposed solution:**
Add optional index support:

```go
type InMemoryConnector[T any, ID comparable] struct {
    data    map[ID]*T
    mu      sync.RWMutex
    getID   func(t *T) ID
    indexes map[string]map[any][]*T // field name -> value -> items
}

// AddIndex creates an index on a specific field
func (r *InMemoryConnector[T, ID]) AddIndex(fieldName string) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if r.indexes == nil {
        r.indexes = make(map[string]map[any][]*T)
    }

    index := make(map[any][]*T)

    for _, item := range r.data {
        v := reflect.ValueOf(item).Elem()
        field := v.FieldByName(fieldName)
        if !field.IsValid() {
            return fmt.Errorf("field %s not found", fieldName)
        }

        value := field.Interface()
        index[value] = append(index[value], item)
    }

    r.indexes[fieldName] = index
    return nil
}

// Updated Query method that uses indexes when available
func (r *InMemoryConnector[T, ID]) Query(_ context.Context, filter *Filter) ([]T, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    // Try to use index if available and filter has single equality condition
    if len(filter.Conditions) == 1 && filter.Conditions[0].Operator == "=" {
        condition := filter.Conditions[0]
        if index, exists := r.indexes[condition.Field]; exists {
            items := index[condition.Value]
            result := make([]T, len(items))
            for i, item := range items {
                result[i] = *item
            }
            return result, nil
        }
    }

    // Fall back to full scan
    var results []T
    for _, item := range r.data {
        if matchesCondition(item, filter) {
            results = append(results, *item)
        }
    }

    return results, nil
}
```

**Additional considerations:**
- Indexes need to be updated on Create/Update/Delete
- Index maintenance adds overhead to write operations
- Only beneficial for read-heavy workloads

**Acceptance criteria:**
- [ ] AddIndex method creates index for specified field
- [ ] Query uses index when applicable
- [ ] Indexes are maintained on Create/Update/Delete
- [ ] Add benchmarks showing performance improvement
- [ ] Document trade-offs

---

## 5. CODE QUALITY IMPROVEMENTS

### CODE-001: Implement logging for transaction errors

**Priority:** MEDIUM
**Files:** `cockroach.go:198-209`, `297-308`, `372-384`

**Problem:**
There are TODO comments for logging rollback and commit errors. These errors should be logged for debugging and monitoring purposes.

**Locations with TODOs:**
- Line 201: `// TODO: Log rollback error: rollbackErr`
- Line 205: `// TODO: Log commit error: commitErr`
- Line 300: `// TODO: Log rollback error: rollbackErr`
- Line 304: `// TODO: Log commit error: commitErr`
- Line 376: `// TODO: Log rollback error: rollbackErr`
- Line 380: `// TODO: Log commit error: commitErr`

**Proposed solution:**

Add a logger interface and optional logger to the connector:

```go
// Logger interface for the connector
type Logger interface {
    Error(msg string, keysAndValues ...interface{})
    Warn(msg string, keysAndValues ...interface{})
}

type CockroachDBConnector[T any, ID comparable] struct {
    pool      *pgxpool.Pool
    tableName string
    getID     func(*T) ID
    columns   []string
    logger    Logger // Optional logger
}

// Update constructor to accept optional logger
func NewCockroachDBConnector[T any, ID comparable](
    pool *pgxpool.Pool,
    tableName string,
    getID func(*T) ID,
    logger Logger, // Can be nil
) (*CockroachDBConnector[T, ID], error) {
    // ... existing validation ...

    return &CockroachDBConnector[T, ID]{
        pool:      pool,
        tableName: tableName,
        getID:     getID,
        columns:   columns,
        logger:    logger,
    }, nil
}

// Helper method for logging
func (r *CockroachDBConnector[T, ID]) logError(msg string, keysAndValues ...interface{}) {
    if r.logger != nil {
        r.logger.Error(msg, keysAndValues...)
    }
}
```

**Update deferred functions:**
```go
defer func() {
    if err != nil {
        if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
            r.logError("failed to rollback transaction", "error", rollbackErr, "table", r.tableName)
        }
    } else {
        if commitErr := tx.Commit(ctx); commitErr != nil {
            r.logError("failed to commit transaction", "error", commitErr, "table", r.tableName)
            err = commitErr
        }
    }
}()
```

**Alternative:** Use standard library `log` or accept an `io.Writer` for logs.

**Acceptance criteria:**
- [ ] Logger interface defined
- [ ] CockroachDBConnector accepts optional logger
- [ ] All TODO comments resolved
- [ ] Backward compatible (logger can be nil)
- [ ] Add example of using connector with logger in documentation

---

### CODE-002: Refactor matchesCondition for better readability

**Priority:** LOW
**File:** `inmemory.go:115-164`

**Problem:**
The `matchesCondition` function is long and has nested logic that makes it hard to read and maintain. The field name capitalization logic is particularly unclear.

**Current code (line 125):**
```go
fieldVal := v.FieldByName(strings.ToTitle(string(condition.Field[0])) + condition.Field[1:])
```

**Proposed refactoring:**

```go
func matchesCondition(item any, filter *Filter) bool {
    v := reflect.ValueOf(item)
    if v.Kind() == reflect.Ptr {
        v = v.Elem()
    }
    if v.Kind() != reflect.Struct {
        return false
    }

    for _, condition := range filter.Conditions {
        if !evaluateSingleCondition(v, condition) {
            return false
        }
    }

    return true
}

func evaluateSingleCondition(structValue reflect.Value, condition Condition) bool {
    fieldVal := getFieldByNameCaseInsensitive(structValue, condition.Field)
    if !fieldVal.IsValid() {
        return false
    }

    valueInterface := fieldVal.Interface()
    return evaluateOperator(valueInterface, condition.Operator, condition.Value)
}

func getFieldByNameCaseInsensitive(v reflect.Value, fieldName string) reflect.Value {
    // Try exact match first
    field := v.FieldByName(fieldName)
    if field.IsValid() {
        return field
    }

    // Try with first letter capitalized (Go convention)
    if len(fieldName) > 0 {
        capitalizedName := strings.ToUpper(string(fieldName[0])) + fieldName[1:]
        field = v.FieldByName(capitalizedName)
        if field.IsValid() {
            return field
        }
    }

    return reflect.Value{}
}

func evaluateOperator(fieldValue any, operator string, conditionValue any) bool {
    switch operator {
    case "=":
        return reflect.DeepEqual(fieldValue, conditionValue)
    case "!=":
        return !reflect.DeepEqual(fieldValue, conditionValue)
    case ">":
        return compare(fieldValue, conditionValue) > 0
    case "<":
        return compare(fieldValue, conditionValue) < 0
    case ">=":
        return compare(fieldValue, conditionValue) >= 0
    case "<=":
        return compare(fieldValue, conditionValue) <= 0
    default:
        return false
    }
}
```

**Acceptance criteria:**
- [ ] Code is split into smaller, focused functions
- [ ] Each function has a single responsibility
- [ ] Field name resolution is clearer
- [ ] All existing tests pass without modification
- [ ] Add unit tests for helper functions

---

### CODE-003: Add package-level and method documentation

**Priority:** LOW
**Files:** All files in sietch package

**Problem:**
Many exported functions and types lack comprehensive documentation comments. Good documentation is essential for a library package.

**Missing documentation:**

1. **Package level** (`repository.go`): Add package doc explaining the overall design
2. **Filter and Condition types** (`filters.go`): Document supported operators
3. **Error variables** (`errors.go`): Document when each error is returned
4. **Connector constructors**: Document parameters and return values
5. **Helper functions**: Document purpose and edge cases

**Example additions:**

```go
// Package sietch provides a unified, generic repository interface for CRUD operations
// across multiple database backends including CockroachDB/PostgreSQL, Redis, and in-memory storage.
//
// The package uses Go generics to provide type-safe repository operations with any entity type T
// and identifier type ID. All implementations follow the Repository[T, ID] interface.
//
// Key Features:
//   - Backend-agnostic CRUD operations
//   - Batch operations with optimizations (transactions for SQL, pipelines for Redis)
//   - Query filtering with support for comparison operators
//   - Thread-safe in-memory implementation
//
// Example:
//
//   type User struct {
//       ID    int64  `db:"id"`
//       Name  string `db:"name"`
//       Email string `db:"email"`
//   }
//
//   repo := sietch.NewInMemoryConnector[User, int64](func(u *User) int64 { return u.ID })
//   user := &User{ID: 1, Name: "John", Email: "john@example.com"}
//   err := repo.Create(context.Background(), user)
//
package sietch
```

```go
// Condition represents a single filter condition for querying.
// It specifies a field name, comparison operator, and value to compare against.
//
// Supported operators:
//   - "=" : Equal
//   - "!=" : Not equal
//   - ">" : Greater than
//   - "<" : Less than
//   - ">=" : Greater than or equal
//   - "<=" : Less than or equal
//
// Example:
//   condition := Condition{
//       Field: "age",
//       Operator: ">=",
//       Value: 18,
//   }
type Condition struct {
    Field    string
    Operator string
    Value    any
}

// Filter groups multiple conditions that are combined with AND logic.
// All conditions must be satisfied for an item to match the filter.
//
// Example:
//   filter := &Filter{
//       Conditions: []Condition{
//           {Field: "age", Operator: ">=", Value: 18},
//           {Field: "country", Operator: "=", Value: "US"},
//       },
//   }
type Filter struct {
    Conditions []Condition
}
```

```go
var (
    // ErrItemNotFound is returned when attempting to get, update, or delete
    // an item that does not exist in the repository.
    ErrItemNotFound = errors.New("item not found")

    // ErrItemAlreadyExists is returned when attempting to create an item
    // with an ID that already exists in the repository.
    ErrItemAlreadyExists = errors.New("item already exists")

    // ErrNoUpdateItem is returned by CockroachDBConnector when an update
    // operation affects zero rows, indicating the item doesn't exist.
    ErrNoUpdateItem = errors.New("no item has been updated")

    // ErrNoDeleteItem is returned by CockroachDBConnector when a delete
    // operation affects zero rows, indicating the item doesn't exist.
    ErrNoDeleteItem = errors.New("no item has been deleted")

    // ErrUnsupportedOperation is returned when attempting an operation
    // that is not supported by the specific connector implementation.
    // For example, RedisConnector does not support Query operations.
    ErrUnsupportedOperation = errors.New("unsupported operation")
)
```

**Acceptance criteria:**
- [ ] Package-level documentation added
- [ ] All exported types have documentation
- [ ] All exported functions have documentation
- [ ] Documentation includes examples where helpful
- [ ] Run `go doc` to verify documentation renders correctly

---

### CODE-004: Add comprehensive benchmarks

**Priority:** LOW
**Files:** Create new files: `inmemory_bench_test.go`, `cockroach_bench_test.go`, `redis_bench_test.go`

**Problem:**
There are no benchmarks to measure performance characteristics of different operations across connectors. Benchmarks are crucial for:
- Comparing connector performance
- Identifying performance regressions
- Understanding scalability characteristics
- Validating optimization efforts

**Benchmarks to create:**

**inmemory_bench_test.go:**
```go
package sietch

import (
    "context"
    "testing"
    "github.com/seb7887/gofw/sietch/internal/testutils"
)

func BenchmarkInMemoryConnector_Create(b *testing.B) {
    repo := NewInMemoryConnector[testutils.Account](func(a *testutils.Account) int64 { return a.ID })
    ctx := context.Background()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        account := testutils.Account{ID: int64(i), Balance: i * 100}
        _ = repo.Create(ctx, &account)
    }
}

func BenchmarkInMemoryConnector_Get(b *testing.B) {
    repo := NewInMemoryConnector[testutils.Account](func(a *testutils.Account) int64 { return a.ID })
    ctx := context.Background()

    // Setup: Create 1000 items
    for i := 0; i < 1000; i++ {
        account := testutils.Account{ID: int64(i), Balance: i * 100}
        _ = repo.Create(ctx, &account)
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = repo.Get(ctx, int64(i%1000))
    }
}

func BenchmarkInMemoryConnector_Query(b *testing.B) {
    repo := NewInMemoryConnector[testutils.Account](func(a *testutils.Account) int64 { return a.ID })
    ctx := context.Background()

    // Setup: Create 10000 items
    for i := 0; i < 10000; i++ {
        account := testutils.Account{ID: int64(i), Balance: i * 100}
        _ = repo.Create(ctx, &account)
    }

    filter := &Filter{
        Conditions: []Condition{
            {Field: "balance", Operator: ">=", Value: 500000},
        },
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = repo.Query(ctx, filter)
    }
}

func BenchmarkInMemoryConnector_BatchCreate(b *testing.B) {
    benchmarkBatchSizes := []int{10, 100, 1000}

    for _, size := range benchmarkBatchSizes {
        b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
            repo := NewInMemoryConnector[testutils.Account](func(a *testutils.Account) int64 { return a.ID })
            ctx := context.Background()

            accounts := make([]testutils.Account, size)
            for i := 0; i < size; i++ {
                accounts[i] = testutils.Account{ID: int64(i), Balance: i * 100}
            }

            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                repo.data = make(map[int64]*testutils.Account) // Reset between iterations
                _ = repo.BatchCreate(ctx, accounts)
            }
        })
    }
}

func BenchmarkInMemoryConnector_ConcurrentReads(b *testing.B) {
    repo := NewInMemoryConnector[testutils.Account](func(a *testutils.Account) int64 { return a.ID })
    ctx := context.Background()

    // Setup: Create 1000 items
    for i := 0; i < 1000; i++ {
        account := testutils.Account{ID: int64(i), Balance: i * 100}
        _ = repo.Create(ctx, &account)
    }

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        i := 0
        for pb.Next() {
            _, _ = repo.Get(ctx, int64(i%1000))
            i++
        }
    })
}
```

**Similar benchmarks for Redis and CockroachDB connectors.**

**Additional useful benchmarks:**
- Memory allocation benchmarks (`-benchmem`)
- Lock contention benchmarks
- Comparison benchmarks (before/after optimization)

**Acceptance criteria:**
- [ ] Benchmarks for all CRUD operations
- [ ] Benchmarks for batch operations with various sizes
- [ ] Concurrent operation benchmarks
- [ ] Memory allocation measurements included
- [ ] README updated with performance characteristics
- [ ] CI configured to track benchmark results over time

---

## Summary

### Priority Distribution:
- **CRITICAL**: 1 fix
- **HIGH**: 4 fixes
- **MEDIUM**: 8 improvements
- **LOW**: 4 improvements

### Category Distribution:
- **Bugs**: 1
- **Validations**: 5
- **Inconsistencies**: 2
- **Features**: 5
- **Code Quality**: 4

### Recommended Implementation Order:
1. BUG-001 (Critical bug fix)
2. VAL-001, VAL-002, VAL-003 (High priority validations)
3. FEAT-002 (SQL injection prevention)
4. VAL-004, VAL-005 (Remaining validations)
5. INC-001, INC-002 (Consistency improvements)
6. FEAT-001, FEAT-004 (Important features)
7. CODE-001 (Logging implementation)
8. FEAT-003, FEAT-005 (Nice-to-have features)
9. CODE-002, CODE-003, CODE-004 (Code quality improvements)

Each item can be tackled independently and used as a prompt for an AI agent to implement the specific improvement or fix.
