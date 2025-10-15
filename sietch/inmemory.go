package sietch

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// InMemoryConnector in-memory implementation of the Repository interface
type InMemoryConnector[T any, ID comparable] struct {
	data  map[ID]*T
	mu    sync.RWMutex
	getID func(t *T) ID // function to extract an element ID
}

func NewInMemoryConnector[T any, ID comparable](getID func(t *T) ID) *InMemoryConnector[T, ID] {
	return &InMemoryConnector[T, ID]{
		data:  make(map[ID]*T),
		getID: getID,
	}
}

func (r *InMemoryConnector[T, ID]) Create(_ context.Context, item *T) error {
	if item == nil {
		return fmt.Errorf("item cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.getID(item)
	if _, exists := r.data[id]; exists {
		return ErrItemAlreadyExists
	}

	r.data[id] = item
	return nil
}

func (r *InMemoryConnector[T, ID]) Get(_ context.Context, id ID) (*T, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, exists := r.data[id]
	if !exists {
		return nil, ErrItemNotFound
	}

	return item, nil
}

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

func (r *InMemoryConnector[T, ID]) Query(_ context.Context, filter *Filter) ([]T, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []T
	for _, item := range r.data {
		if matchesCondition(item, filter) {
			results = append(results, *item)
		}
	}

	// Apply sorting
	if filter != nil && len(filter.Sort) > 0 {
		results = sortResults(results, filter.Sort)
	}

	// Apply DISTINCT
	if filter != nil && filter.Distinct {
		results = distinctResults(results)
	}

	// Apply OFFSET and LIMIT
	if filter != nil {
		if filter.Offset != nil && *filter.Offset > 0 {
			if *filter.Offset >= len(results) {
				return []T{}, nil
			}
			results = results[*filter.Offset:]
		}

		if filter.Limit != nil && *filter.Limit > 0 {
			if *filter.Limit < len(results) {
				results = results[:*filter.Limit]
			}
		}
	}

	return results, nil
}

// Count returns the number of items matching the filter
func (r *InMemoryConnector[T, ID]) Count(_ context.Context, filter *Filter) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var count int64
	for _, item := range r.data {
		if matchesCondition(item, filter) {
			count++
		}
	}

	return count, nil
}

func (r *InMemoryConnector[T, ID]) Update(_ context.Context, item *T) error {
	if item == nil {
		return fmt.Errorf("item cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.getID(item)
	if _, exists := r.data[id]; !exists {
		return ErrItemNotFound
	}

	r.data[id] = item
	return nil
}

func (r *InMemoryConnector[T, ID]) BatchUpdate(ctx context.Context, items []T) error {
	if len(items) == 0 {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, item := range items {
		id := r.getID(&item)
		if _, exists := r.data[id]; !exists {
			return ErrItemNotFound
		}
		r.data[id] = &item
	}
	return nil
}

func (r *InMemoryConnector[T, ID]) Delete(_ context.Context, id ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.data[id]; !exists {
		return ErrItemNotFound
	}

	delete(r.data, id)
	return nil
}

func (r *InMemoryConnector[T, ID]) BatchDelete(ctx context.Context, items []ID) error {
	if len(items) == 0 {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, id := range items {
		if _, exists := r.data[id]; !exists {
			return ErrItemNotFound
		}
		delete(r.data, id)
	}
	return nil
}

func matchesCondition(item any, filter *Filter) bool {
	if filter == nil || len(filter.Conditions) == 0 {
		return true
	}

	// All top-level conditions are ANDed together
	for _, condition := range filter.Conditions {
		if !matchesSingleCondition(item, condition) {
			return false
		}
	}

	return true
}

func matchesSingleCondition(item any, condition Condition) bool {
	// Check if this is a composite condition (logical grouping)
	if condition.IsComposite() {
		return matchesCompositeCondition(item, condition)
	}

	// This is a leaf condition (field comparison)
	return matchesLeafCondition(item, condition)
}

func matchesLeafCondition(item any, condition Condition) bool {
	v := reflect.ValueOf(item)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return false
	}

	fieldVal := v.FieldByName(strings.ToTitle(string(condition.Field[0])) + condition.Field[1:])
	if !fieldVal.IsValid() {
		// field doesn't exist
		return false
	}

	valueInterface := fieldVal.Interface()

	switch condition.Operator {
	case OpEqual:
		return reflect.DeepEqual(valueInterface, condition.Value)
	case OpNotEqual:
		return !reflect.DeepEqual(valueInterface, condition.Value)
	case OpGreaterThan:
		return compare(valueInterface, condition.Value) > 0
	case OpLessThan:
		return compare(valueInterface, condition.Value) < 0
	case OpGreaterThanOrEqual:
		return compare(valueInterface, condition.Value) >= 0
	case OpLessThanOrEqual:
		return compare(valueInterface, condition.Value) <= 0
	case OpIn:
		return inSlice(valueInterface, condition.Value)
	case OpNotIn:
		return !inSlice(valueInterface, condition.Value)
	case OpLike:
		return matchesLike(valueInterface, condition.Value, false)
	case OpILike:
		return matchesLike(valueInterface, condition.Value, true)
	case OpIsNull:
		return fieldVal.IsZero()
	case OpIsNotNull:
		return !fieldVal.IsZero()
	case OpBetween:
		return matchesBetween(valueInterface, condition.Value)
	default:
		// unsupported operator
		return false
	}
}

func matchesCompositeCondition(item any, condition Condition) bool {
	switch condition.LogicalOp {
	case LogicalAND:
		// All nested conditions must be true
		for _, nested := range condition.Conditions {
			if !matchesSingleCondition(item, nested) {
				return false
			}
		}
		return true

	case LogicalOR:
		// At least one nested condition must be true
		for _, nested := range condition.Conditions {
			if matchesSingleCondition(item, nested) {
				return true
			}
		}
		return false

	case LogicalNOT:
		// Negate the result of the nested condition
		if len(condition.Conditions) != 1 {
			return false
		}
		return !matchesSingleCondition(item, condition.Conditions[0])

	default:
		return false
	}
}

// inSlice checks if value is in the slice
func inSlice(value any, sliceValue any) bool {
	slice := reflect.ValueOf(sliceValue)
	if slice.Kind() != reflect.Slice && slice.Kind() != reflect.Array {
		return false
	}

	for i := 0; i < slice.Len(); i++ {
		if reflect.DeepEqual(value, slice.Index(i).Interface()) {
			return true
		}
	}
	return false
}

// matchesLike checks if string matches LIKE pattern
func matchesLike(value any, pattern any, caseInsensitive bool) bool {
	strVal, ok := value.(string)
	if !ok {
		return false
	}
	patternStr, ok := pattern.(string)
	if !ok {
		return false
	}

	if caseInsensitive {
		strVal = strings.ToLower(strVal)
		patternStr = strings.ToLower(patternStr)
	}

	// Simple LIKE implementation: % matches any sequence, _ matches single char
	return matchLikePattern(strVal, patternStr)
}

func matchLikePattern(str, pattern string) bool {
	if pattern == "%" {
		return true
	}
	if pattern == "" {
		return str == ""
	}
	if strings.HasPrefix(pattern, "%") && strings.HasSuffix(pattern, "%") {
		return strings.Contains(str, pattern[1:len(pattern)-1])
	}
	if strings.HasPrefix(pattern, "%") {
		return strings.HasSuffix(str, pattern[1:])
	}
	if strings.HasSuffix(pattern, "%") {
		return strings.HasPrefix(str, pattern[:len(pattern)-1])
	}
	return str == pattern
}

// matchesBetween checks if value is between min and max
func matchesBetween(value any, betweenValue any) bool {
	slice := reflect.ValueOf(betweenValue)
	if slice.Kind() != reflect.Slice && slice.Kind() != reflect.Array {
		return false
	}
	if slice.Len() != 2 {
		return false
	}

	min := slice.Index(0).Interface()
	max := slice.Index(1).Interface()

	return compare(value, min) >= 0 && compare(value, max) <= 0
}

// sortResults sorts the results based on sort fields
func sortResults[T any](results []T, sortFields []SortField) []T {
	if len(sortFields) == 0 {
		return results
	}

	// Create a copy to avoid modifying original
	sorted := make([]T, len(results))
	copy(sorted, results)

	// Sort using multi-field comparison
	sortSlice(sorted, func(a, b *T) bool {
		for _, sf := range sortFields {
			va := reflect.ValueOf(a).Elem()
			vb := reflect.ValueOf(b).Elem()

			fieldA := va.FieldByName(strings.ToTitle(string(sf.Field[0])) + sf.Field[1:])
			fieldB := vb.FieldByName(strings.ToTitle(string(sf.Field[0])) + sf.Field[1:])

			if !fieldA.IsValid() || !fieldB.IsValid() {
				continue
			}

			cmp := compare(fieldA.Interface(), fieldB.Interface())
			if cmp != 0 {
				if sf.Direction == SortAsc {
					return cmp < 0
				}
				return cmp > 0
			}
		}
		return false
	})

	return sorted
}

// sortSlice is a generic sort implementation
func sortSlice[T any](slice []T, less func(a, b *T) bool) {
	n := len(slice)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if less(&slice[j+1], &slice[j]) {
				slice[j], slice[j+1] = slice[j+1], slice[j]
			}
		}
	}
}

// distinctResults removes duplicate items
func distinctResults[T any](results []T) []T {
	if len(results) == 0 {
		return results
	}

	seen := make(map[string]bool)
	var distinct []T

	for _, item := range results {
		// Create a unique key using reflection
		key := fmt.Sprintf("%+v", item)
		if !seen[key] {
			seen[key] = true
			distinct = append(distinct, item)
		}
	}

	return distinct
}

func compare(a, b any) int {
	af, okA := toFloat64(a)
	bf, okB := toFloat64(b)
	if okA && okB {
		if af < bf {
			return -1
		} else if af > bf {
			return 1
		}
		return 0
	}

	// if they are not numeric, we try to compare them as strings
	as, okA := a.(string)
	bs, okB := b.(string)
	if okA && okB {
		if as < bs {
			return -1
		} else if as > bs {
			return 1
		}
		return 0
	}

	return 0 // fallback
}

func toFloat64(v any) (float64, bool) {
	switch t := v.(type) {
	case int:
		return float64(t), true
	case int8:
		return float64(t), true
	case int16:
		return float64(t), true
	case int32:
		return float64(t), true
	case int64:
		return float64(t), true
	case uint:
		return float64(t), true
	case uint8:
		return float64(t), true
	case uint16:
		return float64(t), true
	case uint32:
		return float64(t), true
	case uint64:
		return float64(t), true
	case float32:
		return float64(t), true
	case float64:
		return t, true
	default:
		return 0, false
	}
}

// Exists checks if an entity with the given ID exists
func (r *InMemoryConnector[T, ID]) Exists(_ context.Context, id ID) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.data[id]
	return exists, nil
}

// Upsert creates a new entity or updates an existing one
func (r *InMemoryConnector[T, ID]) Upsert(_ context.Context, item *T) error {
	if item == nil {
		return fmt.Errorf("item cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.getID(item)
	r.data[id] = item
	return nil
}

// BatchUpsert creates or updates multiple entities
func (r *InMemoryConnector[T, ID]) BatchUpsert(_ context.Context, items []T) error {
	if len(items) == 0 {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, item := range items {
		id := r.getID(&item)
		r.data[id] = &item
	}
	return nil
}
