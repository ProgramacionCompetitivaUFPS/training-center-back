package user

import "context"

// TransactionManager handles database transactions for user operations.
// It ensures multiple operations are atomic - either all succeed or all fail.
type TransactionManager interface {
	// WithTx executes the provided function within a database transaction.
	// If fn returns an error, the transaction is rolled back.
	// If fn succeeds, the transaction is committed.
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
