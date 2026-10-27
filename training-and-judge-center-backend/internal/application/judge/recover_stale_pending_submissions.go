package judge

import (
	"context"
	"time"
)

// StalePendingRecoverer re-publishes submissions that have sat in PENDING for
// too long (their queue message was lost) and fails the ones that kept
// getting lost, so no submission is retried forever.
type StalePendingRecoverer interface {
	RecoverStalePending(ctx context.Context, cutoff time.Time) (requeued, failed int, err error)
}

type RecoverStalePendingSubmissionsUseCase struct {
	recoverer  StalePendingRecoverer
	staleAfter time.Duration
}

func NewRecoverStalePendingSubmissionsUseCase(recoverer StalePendingRecoverer, staleAfter time.Duration) *RecoverStalePendingSubmissionsUseCase {
	return &RecoverStalePendingSubmissionsUseCase{recoverer: recoverer, staleAfter: staleAfter}
}

func (uc *RecoverStalePendingSubmissionsUseCase) Execute(ctx context.Context) (requeued, failed int, err error) {
	return uc.recoverer.RecoverStalePending(ctx, time.Now().Add(-uc.staleAfter))
}
