package problem

import (
	"context"
	"errors"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestEnsureNoActiveValidation_NoneFound_ReturnsNil(t *testing.T) {
	validationRepo := &mockValidationRepository{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			return nil, false, nil
		},
	}
	if err := ensureNoActiveValidation(context.Background(), validationRepo, testProbID); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEnsureNoActiveValidation_Running_ReturnsConflict(t *testing.T) {
	validationRepo := &mockValidationRepository{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			return runningValidationFixture(), true, nil
		},
	}
	err := ensureNoActiveValidation(context.Background(), validationRepo, testProbID)
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainProblem.ErrCodeValidationInProgress {
		t.Errorf("expected ErrCodeValidationInProgress, got %v", err)
	}
}

func TestEnsureNoActiveValidation_Pending_ReturnsConflict(t *testing.T) {
	validationRepo := &mockValidationRepository{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			return newValidationFixture(), true, nil // PENDING, not yet Start()ed
		},
	}
	err := ensureNoActiveValidation(context.Background(), validationRepo, testProbID)
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainProblem.ErrCodeValidationInProgress {
		t.Errorf("expected ErrCodeValidationInProgress, got %v", err)
	}
}

func TestEnsureNoActiveValidation_TerminalValidation_ReturnsNil(t *testing.T) {
	v := runningValidationFixture()
	if err := v.MarkPassed(`{}`, testNow); err != nil {
		t.Fatalf("fixture MarkPassed: %v", err)
	}
	validationRepo := &mockValidationRepository{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			return v, true, nil
		},
	}
	if err := ensureNoActiveValidation(context.Background(), validationRepo, testProbID); err != nil {
		t.Errorf("expected no error for a terminal (PASSED) validation, got %v", err)
	}
}

func TestEnsureNoActiveValidation_RepositoryError_Propagates(t *testing.T) {
	wantErr := apperror.NewInternal()
	validationRepo := &mockValidationRepository{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			return nil, false, wantErr
		},
	}
	if err := ensureNoActiveValidation(context.Background(), validationRepo, testProbID); !errors.Is(err, wantErr) {
		t.Errorf("expected the repository error to propagate, got %v", err)
	}
}
