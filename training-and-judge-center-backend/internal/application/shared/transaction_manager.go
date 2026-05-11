package shared

import "context"

type TransactionManager interface {
	WithTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
