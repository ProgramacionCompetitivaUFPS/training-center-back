package problem

import (
	"context"
	"time"
)

type StaleValidationRecoverer interface {
	RecoverStaleBefore(ctx context.Context, cutoff time.Time) (int, error)
}
