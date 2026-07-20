package judge

import (
	"context"
	"time"
)

type StaleSubmissionRecoverer interface {
	RecoverStaleBefore(ctx context.Context, cutoff time.Time) (int, error)
}
