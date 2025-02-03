package sietch

import (
	"context"
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
	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.getID(item)
	if _, exists := r.data[id]; exists {
		return ErrItemNotFound
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
	for _, item := range items {
		if err := r.Create(ctx, &item); err != nil {
			return err
		}
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

	return results, nil
}

func (r *InMemoryConnector[T, ID]) Update(_ context.Context, item *T) error {
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
	for _, item := range items {
		if err := r.Update(ctx, &item); err != nil {
			return err
		}
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
	for _, item := range items {
		if err := r.Delete(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

func matchesCondition(item any, filter *Filter) bool {
	v := reflect.ValueOf(item)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return false
	}

	for _, condition := range filter.Conditions {
		fieldVal := v.FieldByName(strings.ToTitle(string(condition.Field[0])) + condition.Field[1:])
		if !fieldVal.IsValid() {
			// field doesn't exist
			return false
		}

		valueInterface := fieldVal.Interface()
		switch condition.Operator {
		case "=":
			if !reflect.DeepEqual(valueInterface, condition.Value) {
				return false
			}
		case "!=":
			if reflect.DeepEqual(valueInterface, condition.Value) {
				return false
			}
		case ">":
			if compare(valueInterface, condition.Value) <= 0 {
				return false
			}
		case "<":
			if compare(valueInterface, condition.Value) >= 0 {
				return false
			}
		case ">=":
			if compare(valueInterface, condition.Value) < 0 {
				return false
			}
		case "<=":
			if compare(valueInterface, condition.Value) > 0 {
				return false
			}
		default:
			// unsupported operator
			return false
		}
	}

	return true
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
