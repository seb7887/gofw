package sietch

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TransactionManager manages database transactions across multiple repositories
type TransactionManager struct {
	pool *pgxpool.Pool
}

// MultiRepoTxFunc is a function that executes operations within a transaction context
// All operations should use the provided context which contains the active transaction
type MultiRepoTxFunc func(ctx context.Context) error

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(pool *pgxpool.Pool) *TransactionManager {
	if pool == nil {
		panic("pool cannot be nil")
	}
	return &TransactionManager{pool: pool}
}

// WithTx executes the provided function within a transaction
// If the function returns an error, the transaction is rolled back
// If the function completes successfully, the transaction is committed
// The transaction is also rolled back if a panic occurs
func (tm *TransactionManager) WithTx(ctx context.Context, fn MultiRepoTxFunc) error {
	// Begin transaction
	tx, err := tm.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Setup panic recovery and defer rollback/commit
	defer func() {
		if p := recover(); p != nil {
			// Rollback on panic
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				// Log rollback error (in production, use proper logging)
				fmt.Printf("rollback after panic failed: %v\n", rbErr)
			}
			// Re-raise the panic
			panic(p)
		}
	}()

	// Inject transaction into context
	txCtx := context.WithValue(ctx, txKey{}, tx)

	// Execute the function
	err = fn(txCtx)
	if err != nil {
		// Rollback on error
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx error: %w, rollback error: %v", err, rbErr)
		}
		return err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// txKey is the context key type for transaction injection
type txKey struct{}

// getTxFromContext extracts the transaction from context, if present
func getTxFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	return tx, ok
}
