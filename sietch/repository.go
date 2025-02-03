package sietch

import "context"

// Repository defines a generic contract for CRUD operations
// T represents the entity type and ID the identifier type
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
