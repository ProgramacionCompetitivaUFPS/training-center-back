package problem

import (
	"context"
	"time"

	"github.com/training-judge-center/backend/pkg/apperror"
)

// defaultAwaitPollInterval and defaultAwaitPollTimeout are the values
// NewAwaitProblemValidationUseCase sets in production. They are unexported
// struct fields (not used directly) so tests can build an
// AwaitProblemValidationUseCase with shorter values instead of waiting on
// real wall-clock time.
const (
	defaultAwaitPollInterval = 750 * time.Millisecond
	defaultAwaitPollTimeout  = 9 * time.Minute
)

type AwaitProblemValidationInput struct {
	ValidationID string
}

type AwaitProblemValidationUseCase struct {
	statusCheck  *GetProblemValidationStatusUseCase
	pollInterval time.Duration
	pollTimeout  time.Duration
}

func NewAwaitProblemValidationUseCase(statusCheck *GetProblemValidationStatusUseCase) *AwaitProblemValidationUseCase {
	return &AwaitProblemValidationUseCase{
		statusCheck:  statusCheck,
		pollInterval: defaultAwaitPollInterval,
		pollTimeout:  defaultAwaitPollTimeout,
	}
}

// Execute blocks until the validation reaches a final state or uc.pollTimeout
// elapses. The worker keeps working regardless of whether anyone is still
// waiting — a retried publish call will either find the same result already
// there, or reuse the still-running attempt.
//
// If ctx is already canceled when the timeout fires, that cancellation is
// returned as-is (the caller decided to stop waiting, e.g. an HTTP client
// disconnected) instead of being reported as a timeout.
func (uc *AwaitProblemValidationUseCase) Execute(ctx context.Context, in AwaitProblemValidationInput) (*GetProblemValidationStatusOutput, error) {
	pollCtx, cancel := context.WithTimeout(ctx, uc.pollTimeout)
	defer cancel()

	ticker := time.NewTicker(uc.pollInterval)
	defer ticker.Stop()

	for {
		out, err := uc.statusCheck.Execute(pollCtx, GetProblemValidationStatusInput{ValidationID: in.ValidationID})
		if err != nil {
			return nil, err
		}
		if out.Terminal {
			return out, nil
		}

		select {
		case <-pollCtx.Done():
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, apperror.NewServiceUnavailable(ErrCodeValidationTimedOut, "Validation is taking longer than expected; check back later")
		case <-ticker.C:
		}
	}
}
