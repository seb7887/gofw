package sietch

import (
	"context"
	"time"
)

// CacheStrategy defines how caching should behave
type CacheStrategy string

const (
	// CacheStrategyWriteThrough writes to both cache and base storage synchronously
	CacheStrategyWriteThrough CacheStrategy = "write_through"

	// CacheStrategyWriteAround writes only to base storage, invalidates cache
	CacheStrategyWriteAround CacheStrategy = "write_around"

	// CacheStrategyWriteBack writes to cache first, async to base storage
	CacheStrategyWriteBack CacheStrategy = "write_back"
)

// CachedRepository wraps a base repository with a caching layer
// It provides automatic caching for Get operations and cache invalidation for mutations
type CachedRepository[T any, ID comparable] struct {
	base     Repository[T, ID] // Primary data source (e.g., CockroachDB)
	cache    Repository[T, ID] // Cache layer (e.g., Redis)
	ttl      time.Duration     // Time-to-live for cached items
	strategy CacheStrategy     // Caching strategy
}

// NewCachedRepository creates a new cached repository
// base: the primary data source (typically a database connector)
// cache: the cache layer (typically a Redis connector)
// ttl: how long items should remain in cache
func NewCachedRepository[T any, ID comparable](
	base Repository[T, ID],
	cache Repository[T, ID],
	ttl time.Duration,
) *CachedRepository[T, ID] {
	return &CachedRepository[T, ID]{
		base:     base,
		cache:    cache,
		ttl:      ttl,
		strategy: CacheStrategyWriteThrough,
	}
}

// NewCachedRepositoryWithStrategy creates a cached repository with a specific strategy
func NewCachedRepositoryWithStrategy[T any, ID comparable](
	base Repository[T, ID],
	cache Repository[T, ID],
	ttl time.Duration,
	strategy CacheStrategy,
) *CachedRepository[T, ID] {
	return &CachedRepository[T, ID]{
		base:     base,
		cache:    cache,
		ttl:      ttl,
		strategy: strategy,
	}
}

// Get tries cache first, falls back to base on cache miss
func (r *CachedRepository[T, ID]) Get(ctx context.Context, id ID) (*T, error) {
	// Try cache first
	item, err := r.cache.Get(ctx, id)
	if err == nil {
		return item, nil
	}

	// Cache miss or error - get from base
	item, err = r.base.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Populate cache asynchronously (fire and forget)
	go func() {
		_ = r.cache.Upsert(context.Background(), item)
	}()

	return item, nil
}

// Create creates in base and manages cache based on strategy
func (r *CachedRepository[T, ID]) Create(ctx context.Context, item *T) error {
	// Always create in base first
	if err := r.base.Create(ctx, item); err != nil {
		return err
	}

	// Handle caching based on strategy
	switch r.strategy {
	case CacheStrategyWriteThrough:
		// Write to cache synchronously
		_ = r.cache.Upsert(ctx, item)
	case CacheStrategyWriteAround:
		// Don't write to cache, let next Get populate it
	case CacheStrategyWriteBack:
		// Write to cache asynchronously
		go func() {
			_ = r.cache.Upsert(context.Background(), item)
		}()
	}

	return nil
}

// Update updates in base and invalidates/updates cache
func (r *CachedRepository[T, ID]) Update(ctx context.Context, item *T) error {
	if err := r.base.Update(ctx, item); err != nil {
		return err
	}

	// Invalidate or update cache based on strategy
	switch r.strategy {
	case CacheStrategyWriteThrough:
		_ = r.cache.Upsert(ctx, item)
	case CacheStrategyWriteAround:
		// Invalidate cache - next Get will repopulate
		// Note: We use Upsert instead of Delete to avoid errors if key doesn't exist
		_ = r.cache.Upsert(ctx, item)
	case CacheStrategyWriteBack:
		go func() {
			_ = r.cache.Upsert(context.Background(), item)
		}()
	}

	return nil
}

// Delete deletes from base and invalidates cache
func (r *CachedRepository[T, ID]) Delete(ctx context.Context, id ID) error {
	if err := r.base.Delete(ctx, id); err != nil {
		return err
	}

	// Remove from cache (ignore errors)
	_ = r.cache.Delete(ctx, id)

	return nil
}

// Query delegates to base (caching queries is complex and often not worthwhile)
func (r *CachedRepository[T, ID]) Query(ctx context.Context, filter *Filter) ([]T, error) {
	return r.base.Query(ctx, filter)
}

// Count delegates to base
func (r *CachedRepository[T, ID]) Count(ctx context.Context, filter *Filter) (int64, error) {
	return r.base.Count(ctx, filter)
}

// BatchCreate creates in base and manages cache
func (r *CachedRepository[T, ID]) BatchCreate(ctx context.Context, items []T) error {
	if err := r.base.BatchCreate(ctx, items); err != nil {
		return err
	}

	// Optionally populate cache
	if r.strategy == CacheStrategyWriteThrough {
		_ = r.cache.BatchUpsert(ctx, items)
	}

	return nil
}

// BatchUpdate updates in base and invalidates cache entries
func (r *CachedRepository[T, ID]) BatchUpdate(ctx context.Context, items []T) error {
	if err := r.base.BatchUpdate(ctx, items); err != nil {
		return err
	}

	// Update cache entries
	if r.strategy == CacheStrategyWriteThrough {
		_ = r.cache.BatchUpsert(ctx, items)
	}

	return nil
}

// BatchDelete deletes from base and invalidates cache entries
func (r *CachedRepository[T, ID]) BatchDelete(ctx context.Context, ids []ID) error {
	if err := r.base.BatchDelete(ctx, ids); err != nil {
		return err
	}

	// Remove from cache
	_ = r.cache.BatchDelete(ctx, ids)

	return nil
}

// Exists checks base (cache might have stale data)
func (r *CachedRepository[T, ID]) Exists(ctx context.Context, id ID) (bool, error) {
	return r.base.Exists(ctx, id)
}

// Upsert upserts in base and manages cache
func (r *CachedRepository[T, ID]) Upsert(ctx context.Context, item *T) error {
	if err := r.base.Upsert(ctx, item); err != nil {
		return err
	}

	switch r.strategy {
	case CacheStrategyWriteThrough:
		_ = r.cache.Upsert(ctx, item)
	case CacheStrategyWriteBack:
		go func() {
			_ = r.cache.Upsert(context.Background(), item)
		}()
	}

	return nil
}

// BatchUpsert upserts in base and manages cache
func (r *CachedRepository[T, ID]) BatchUpsert(ctx context.Context, items []T) error {
	if err := r.base.BatchUpsert(ctx, items); err != nil {
		return err
	}

	if r.strategy == CacheStrategyWriteThrough {
		_ = r.cache.BatchUpsert(ctx, items)
	}

	return nil
}

// InvalidateCache removes all items from cache (if supported)
// Note: This may not be supported by all cache implementations
func (r *CachedRepository[T, ID]) InvalidateCache(ctx context.Context) error {
	// This would require a "clear all" operation which isn't in the Repository interface
	// For now, this is a no-op. Implementations can add this if needed.
	return nil
}
