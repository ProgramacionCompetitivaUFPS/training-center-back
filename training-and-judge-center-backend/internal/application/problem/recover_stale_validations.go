package problem

import (
	"context"
	"time"
)

type RecoverStaleValidationsUseCase struct {
	recoverer  StaleValidationRecoverer
	staleAfter time.Duration
}

func NewRecoverStaleValidationsUseCase(recoverer StaleValidationRecoverer, staleAfter time.Duration) *RecoverStaleValidationsUseCase {
	return &RecoverStaleValidationsUseCase{recoverer: recoverer, staleAfter: staleAfter}
}

func (uc *RecoverStaleValidationsUseCase) Execute(ctx context.Context) (int, error) {
	cutoff := time.Now().Add(-uc.staleAfter)
	return uc.recoverer.RecoverStaleBefore(ctx, cutoff)
}
