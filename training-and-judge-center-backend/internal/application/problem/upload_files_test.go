package problem

import (
	"context"
	"errors"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// TestUploadProblemFiles_ActiveValidation_ReturnsConflict only checks that
// the guard runs before any file-type-specific logic — zipParser is nil and
// never reached, since the conflict short-circuits before the file-type
// switch. The rest of this use case's behavior has no test coverage yet
// (pre-existing gap, out of scope for closing this race condition).
func TestUploadProblemFiles_ActiveValidation_ReturnsConflict(t *testing.T) {
	repo := repoWith(newDraftProblem())
	validationRepo := &mockValidationRepository{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			return runningValidationFixture(), true, nil
		},
	}
	uc := NewUploadProblemFilesUseCase(repo, validationRepo, &mockFileStorage{}, nil, newDefaultSettings())

	_, err := uc.Execute(context.Background(), UploadProblemFilesInput{
		Slug:        testSlug,
		FileType:    FileTypeSolution,
		FileName:    "sol.cpp",
		FileData:    []byte("int main(){}"),
		CurrentUser: asCoach(authorID),
	})
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainProblem.ErrCodeValidationInProgress {
		t.Errorf("expected ErrCodeValidationInProgress, got %v", err)
	}
}
