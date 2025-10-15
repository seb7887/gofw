package sietch

import (
	"context"
	"fmt"
)

// WithTx executes the given function within a transaction simulation.
// For InMemory connector, this creates a snapshot of the data, executes the function,
// and either commits (keeps changes) or rollbacks (restores snapshot) based on the result.
func (r *InMemoryConnector[T, ID]) WithTx(ctx context.Context, fn TxFunc[T, ID]) error {
	r.mu.Lock()

	// Create snapshot of current data
	snapshot := make(map[ID]*T)
	for k, v := range r.data {
		// Create a copy of the value
		copyValue := *v
		snapshot[k] = &copyValue
	}
	r.mu.Unlock()

	// Defer rollback in case of panic
	defer func() {
		if p := recover(); p != nil {
			r.mu.Lock()
			// Restore from snapshot
			r.data = snapshot
			r.mu.Unlock()
			panic(p)
		}
	}()

	// Execute the user function
	err := fn(r)
	if err != nil {
		r.mu.Lock()
		// Rollback: restore from snapshot
		r.data = snapshot
		r.mu.Unlock()
		return fmt.Errorf("tx error: %w", err)
	}

	// Commit: changes are already in r.data, just discard snapshot
	return nil
}
