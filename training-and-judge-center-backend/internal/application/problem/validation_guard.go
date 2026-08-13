package problem

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ensureNoActiveValidation rejects modifying a problem's files while a
// validation ticket is PENDING/RUNNING for it — UpdateProblem,
// UploadProblemFiles, and DeleteProblemFile all call this before touching
// anything, so a re-uploaded checker (or a changed limit) can't silently
// corrupt a validation the worker is compiling/running right now.
func ensureNoActiveValidation(ctx context.Context, validationRepo problem.ProblemValidationRepository, problemID string) error {
	existing, found, err := validationRepo.FindLatestByProblemID(ctx, problemID)
	if err != nil {
		return err
	}
	if found && !existing.Status().IsFinal() {
		return apperror.NewConflict(problem.ErrCodeValidationInProgress, "Cannot modify problem files while a validation is in progress")
	}
	return nil
}
