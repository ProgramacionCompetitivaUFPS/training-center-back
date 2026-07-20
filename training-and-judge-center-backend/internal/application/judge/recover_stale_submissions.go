package judge

import (
	"context"
	"time"
)

type RecoverStaleSubmissionsUseCase struct {
	recoverer  StaleSubmissionRecoverer
	staleAfter time.Duration
}

func NewRecoverStaleSubmissionsUseCase(recoverer StaleSubmissionRecoverer, staleAfter time.Duration) *RecoverStaleSubmissionsUseCase {
	return &RecoverStaleSubmissionsUseCase{recoverer: recoverer, staleAfter: staleAfter}
}

func (uc *RecoverStaleSubmissionsUseCase) Execute(ctx context.Context) (int, error) {
	cutoff := time.Now().Add(-uc.staleAfter)
	return uc.recoverer.RecoverStaleBefore(ctx, cutoff)
}
