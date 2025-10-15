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
	Count(ctx context.Context, filter *Filter) (int64, error)

	// Exists checks if an entity with the given ID exists
	Exists(ctx context.Context, id ID) (bool, error)

	// Upsert creates a new entity or updates an existing one
	Upsert(ctx context.Context, item *T) error

	// BatchUpsert creates or updates multiple entities
	BatchUpsert(ctx context.Context, items []T) error
}

// TxFunc is a function that operates within a transaction context
type TxFunc[T any, ID comparable] func(repo Repository[T, ID]) error

// Transactional defines an optional interface for transaction support
// Implementations can use type assertion to check if a repository supports transactions:
//   if txRepo, ok := repo.(Transactional[T, ID]); ok { ... }
type Transactional[T any, ID comparable] interface {
	// WithTx executes the given function within a transaction.
	// If the function returns an error, the transaction is rolled back.
	// If the function returns nil, the transaction is committed.
	// If the function panics, the transaction is rolled back and the panic is re-raised.
	WithTx(ctx context.Context, fn TxFunc[T, ID]) error
}
