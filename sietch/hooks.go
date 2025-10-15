package sietch

import "context"

// Hook defines lifecycle callbacks for repository operations
// Implementations can intercept and react to repository events
type Hook[T any, ID comparable] interface {
	// BeforeCreate is called before creating a new entity
	// Return error to abort the operation
	BeforeCreate(ctx context.Context, item *T) error

	// AfterCreate is called after successfully creating an entity
	// Errors are logged but don't affect the operation result
	AfterCreate(ctx context.Context, item *T) error

	// BeforeUpdate is called before updating an entity
	// Return error to abort the operation
	BeforeUpdate(ctx context.Context, item *T) error

	// AfterUpdate is called after successfully updating an entity
	// Errors are logged but don't affect the operation result
	AfterUpdate(ctx context.Context, item *T) error

	// BeforeDelete is called before deleting an entity
	// Return error to abort the operation
	BeforeDelete(ctx context.Context, id ID) error

	// AfterDelete is called after successfully deleting an entity
	// Errors are logged but don't affect the operation result
	AfterDelete(ctx context.Context, id ID) error

	// BeforeQuery is called before executing a query
	// Can modify the filter before execution
	BeforeQuery(ctx context.Context, filter *Filter) error

	// AfterQuery is called after successfully executing a query
	// Errors are logged but don't affect the operation result
	AfterQuery(ctx context.Context, results []T) error
}

// BaseHook provides a default implementation of Hook interface
// Embed this in custom hooks to only implement needed methods
type BaseHook[T any, ID comparable] struct{}

func (h *BaseHook[T, ID]) BeforeCreate(ctx context.Context, item *T) error { return nil }
func (h *BaseHook[T, ID]) AfterCreate(ctx context.Context, item *T) error  { return nil }
func (h *BaseHook[T, ID]) BeforeUpdate(ctx context.Context, item *T) error { return nil }
func (h *BaseHook[T, ID]) AfterUpdate(ctx context.Context, item *T) error  { return nil }
func (h *BaseHook[T, ID]) BeforeDelete(ctx context.Context, id ID) error   { return nil }
func (h *BaseHook[T, ID]) AfterDelete(ctx context.Context, id ID) error    { return nil }
func (h *BaseHook[T, ID]) BeforeQuery(ctx context.Context, filter *Filter) error {
	return nil
}
func (h *BaseHook[T, ID]) AfterQuery(ctx context.Context, results []T) error { return nil }

// HookRegistry manages a collection of hooks
type HookRegistry[T any, ID comparable] struct {
	hooks []Hook[T, ID]
}

// NewHookRegistry creates a new hook registry
func NewHookRegistry[T any, ID comparable]() *HookRegistry[T, ID] {
	return &HookRegistry[T, ID]{
		hooks: make([]Hook[T, ID], 0),
	}
}

// AddHook registers a new hook
func (r *HookRegistry[T, ID]) AddHook(hook Hook[T, ID]) {
	r.hooks = append(r.hooks, hook)
}

// RemoveAllHooks clears all registered hooks
func (r *HookRegistry[T, ID]) RemoveAllHooks() {
	r.hooks = make([]Hook[T, ID], 0)
}

// ExecuteBeforeCreate runs all BeforeCreate hooks
func (r *HookRegistry[T, ID]) ExecuteBeforeCreate(ctx context.Context, item *T) error {
	for _, hook := range r.hooks {
		if err := hook.BeforeCreate(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteAfterCreate runs all AfterCreate hooks (errors are collected but don't stop execution)
func (r *HookRegistry[T, ID]) ExecuteAfterCreate(ctx context.Context, item *T) error {
	var firstErr error
	for _, hook := range r.hooks {
		if err := hook.AfterCreate(ctx, item); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// ExecuteBeforeUpdate runs all BeforeUpdate hooks
func (r *HookRegistry[T, ID]) ExecuteBeforeUpdate(ctx context.Context, item *T) error {
	for _, hook := range r.hooks {
		if err := hook.BeforeUpdate(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteAfterUpdate runs all AfterUpdate hooks
func (r *HookRegistry[T, ID]) ExecuteAfterUpdate(ctx context.Context, item *T) error {
	var firstErr error
	for _, hook := range r.hooks {
		if err := hook.AfterUpdate(ctx, item); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// ExecuteBeforeDelete runs all BeforeDelete hooks
func (r *HookRegistry[T, ID]) ExecuteBeforeDelete(ctx context.Context, id ID) error {
	for _, hook := range r.hooks {
		if err := hook.BeforeDelete(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteAfterDelete runs all AfterDelete hooks
func (r *HookRegistry[T, ID]) ExecuteAfterDelete(ctx context.Context, id ID) error {
	var firstErr error
	for _, hook := range r.hooks {
		if err := hook.AfterDelete(ctx, id); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// ExecuteBeforeQuery runs all BeforeQuery hooks
func (r *HookRegistry[T, ID]) ExecuteBeforeQuery(ctx context.Context, filter *Filter) error {
	for _, hook := range r.hooks {
		if err := hook.BeforeQuery(ctx, filter); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteAfterQuery runs all AfterQuery hooks
func (r *HookRegistry[T, ID]) ExecuteAfterQuery(ctx context.Context, results []T) error {
	var firstErr error
	for _, hook := range r.hooks {
		if err := hook.AfterQuery(ctx, results); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Hookable is an optional interface that repositories can implement to support hooks
type Hookable[T any, ID comparable] interface {
	// AddHook registers a hook with the repository
	AddHook(hook Hook[T, ID])

	// RemoveAllHooks clears all hooks
	RemoveAllHooks()
}
