package problem

import (
	"context"
	"errors"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// TestDeleteProblemFile_ActiveValidation_ReturnsConflict only checks that
// the guard runs before any file-type-specific logic. The rest of this use
// case's behavior has no test coverage yet (pre-existing gap, out of scope
// for closing this race condition).
func TestDeleteProblemFile_ActiveValidation_ReturnsConflict(t *testing.T) {
	repo := repoWith(newDraftProblem())
	validationRepo := &mockValidationRepository{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			return runningValidationFixture(), true, nil
		},
	}
	uc := NewDeleteProblemFileUseCase(repo, validationRepo, &mockFileStorage{})

	err := uc.Execute(context.Background(), DeleteProblemFileInput{
		Slug:        testSlug,
		FileType:    FileTypeChecker,
		CurrentUser: asCoach(authorID),
	})
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainProblem.ErrCodeValidationInProgress {
		t.Errorf("expected ErrCodeValidationInProgress, got %v", err)
	}
}
