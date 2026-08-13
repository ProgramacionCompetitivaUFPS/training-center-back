package problem

import (
	"context"
	"errors"
	"testing"
	"time"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func runningValidationFixture() *domainProblem.ProblemValidation {
	v := newValidationFixture()
	if err := v.Start(testNow); err != nil {
		panic(err)
	}
	return v
}

func newAwaitUseCase(validationRepo *mockValidationRepository, statusProvider *mockProblemStatusProvider, pollInterval, pollTimeout time.Duration) *AwaitProblemValidationUseCase {
	return &AwaitProblemValidationUseCase{
		statusCheck:  NewGetProblemValidationStatusUseCase(validationRepo, statusProvider),
		pollInterval: pollInterval,
		pollTimeout:  pollTimeout,
	}
}

func TestAwaitProblemValidation_ImmediatelyTerminal_ReturnsResult(t *testing.T) {
	v := runningValidationFixture()
	if err := v.MarkPassed(`{"validationLogs":["ok"]}`, testNow); err != nil {
		t.Fatalf("fixture MarkPassed: %v", err)
	}
	uc := NewAwaitProblemValidationUseCase(NewGetProblemValidationStatusUseCase(repoReturning(v), &mockProblemStatusProvider{}))

	out, err := uc.Execute(context.Background(), AwaitProblemValidationInput{ValidationID: "validation-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Terminal || !out.Passed {
		t.Errorf("expected a terminal, passed result, got %+v", out)
	}
}

func TestAwaitProblemValidation_PollsUntilTerminal(t *testing.T) {
	checks := 0
	validationRepo := &mockValidationRepository{
		findByIDFn: func(_ context.Context, id string) (*domainProblem.ProblemValidation, error) {
			checks++
			if checks < 3 {
				return runningValidationFixture(), nil
			}
			v := runningValidationFixture()
			if err := v.MarkPassed(`{"validationLogs":["ok"]}`, testNow); err != nil {
				t.Fatalf("fixture MarkPassed: %v", err)
			}
			return v, nil
		},
	}
	uc := newAwaitUseCase(validationRepo, &mockProblemStatusProvider{}, 5*time.Millisecond, 500*time.Millisecond)

	out, err := uc.Execute(context.Background(), AwaitProblemValidationInput{ValidationID: "validation-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Terminal {
		t.Error("expected a terminal result")
	}
	if checks < 3 {
		t.Errorf("expected at least 3 status checks before resolving, got %d", checks)
	}
}

func TestAwaitProblemValidation_TimesOut_ReturnsServiceUnavailable(t *testing.T) {
	validationRepo := &mockValidationRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, error) {
			return runningValidationFixture(), nil // never becomes terminal
		},
	}
	uc := newAwaitUseCase(validationRepo, &mockProblemStatusProvider{}, 5*time.Millisecond, 20*time.Millisecond)

	_, err := uc.Execute(context.Background(), AwaitProblemValidationInput{ValidationID: "validation-1"})
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeValidationTimedOut || appErr.Kind != apperror.KindServiceUnavailable {
		t.Errorf("expected a service-unavailable timeout error, got %v", err)
	}
}

func TestAwaitProblemValidation_ParentContextCanceled_ReturnsContextError(t *testing.T) {
	validationRepo := &mockValidationRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, error) {
			return runningValidationFixture(), nil // never becomes terminal
		},
	}
	uc := newAwaitUseCase(validationRepo, &mockProblemStatusProvider{}, 5*time.Millisecond, 500*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // simulate the caller (e.g. an HTTP handler) giving up before the timeout

	_, err := uc.Execute(ctx, AwaitProblemValidationInput{ValidationID: "validation-1"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestAwaitProblemValidation_StatusCheckError_Propagates(t *testing.T) {
	wantErr := apperror.NewNotFound("VALIDATION_NOT_FOUND", "validation not found")
	validationRepo := &mockValidationRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, error) {
			return nil, wantErr
		},
	}
	uc := NewAwaitProblemValidationUseCase(NewGetProblemValidationStatusUseCase(validationRepo, &mockProblemStatusProvider{}))

	_, err := uc.Execute(context.Background(), AwaitProblemValidationInput{ValidationID: "missing"})
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != "VALIDATION_NOT_FOUND" {
		t.Errorf("expected the status-check error to propagate, got %v", err)
	}
}
