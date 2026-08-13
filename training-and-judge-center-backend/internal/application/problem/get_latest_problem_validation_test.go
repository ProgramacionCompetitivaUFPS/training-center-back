package problem

import (
	"context"
	"errors"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newGetLatestProblemValidationUseCase(repo *mockProblemRepository, validationRepo *mockValidationRepository, statusProvider *mockProblemStatusProvider) *GetLatestProblemValidationUseCase {
	statusCheck := NewGetProblemValidationStatusUseCase(validationRepo, statusProvider)
	return NewGetLatestProblemValidationUseCase(repo, validationRepo, statusCheck)
}

func TestGetLatestProblemValidation_NoAttemptsYet_ReturnsFoundFalse(t *testing.T) {
	repo := repoWith(newDraftProblem())
	validationRepo := &mockValidationRepository{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			return nil, false, nil
		},
	}
	uc := newGetLatestProblemValidationUseCase(repo, validationRepo, &mockProblemStatusProvider{})

	out, err := uc.Execute(context.Background(), GetLatestProblemValidationInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Found {
		t.Error("expected Found=false when the problem never had a validation attempt")
	}
}

func TestGetLatestProblemValidation_ProblemNotFound_Propagates(t *testing.T) {
	repo := &mockProblemRepository{
		findBySlugFn: func(_ context.Context, _ domainProblem.Slug) (*domainProblem.Problem, error) {
			return nil, apperror.NewNotFound("PROBLEM_NOT_FOUND", "not found")
		},
	}
	uc := newGetLatestProblemValidationUseCase(repo, &mockValidationRepository{}, &mockProblemStatusProvider{})

	_, err := uc.Execute(context.Background(), GetLatestProblemValidationInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != "PROBLEM_NOT_FOUND" {
		t.Errorf("expected the repository error to propagate, got %v", err)
	}
}

func TestGetLatestProblemValidation_Forbidden_Stranger(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := newGetLatestProblemValidationUseCase(repo, &mockValidationRepository{}, &mockProblemStatusProvider{})

	_, err := uc.Execute(context.Background(), GetLatestProblemValidationInput{Slug: testSlug, CurrentUser: asContestant(strangerID)})
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindForbidden {
		t.Errorf("expected Forbidden, got %v", err)
	}
}

func TestGetLatestProblemValidation_Found_ReturnsDecodedStatus(t *testing.T) {
	repo := repoWith(newDraftProblem())
	v := runningValidationFixture()
	if err := v.MarkPassed(`{"validationLogs":["ok"]}`, testNow); err != nil {
		t.Fatalf("fixture MarkPassed: %v", err)
	}
	validationRepo := &mockValidationRepository{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			return v, true, nil
		},
		findByIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, error) {
			return v, nil
		},
	}
	statusProvider := &mockProblemStatusProvider{
		getStatusFn: func(_ context.Context, _ string) (string, error) { return "PUBLISHED", nil },
	}
	uc := newGetLatestProblemValidationUseCase(repo, validationRepo, statusProvider)

	out, err := uc.Execute(context.Background(), GetLatestProblemValidationInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Found || !out.Terminal || !out.Passed || out.Status != "PUBLISHED" {
		t.Errorf("out: got %+v", out)
	}
	if len(out.Report.ValidationLogs) != 1 {
		t.Errorf("ValidationLogs: got %v", out.Report.ValidationLogs)
	}
}

func TestGetLatestProblemValidation_StillRunning_ReturnsNotTerminal(t *testing.T) {
	repo := repoWith(newDraftProblem())
	v := runningValidationFixture()
	validationRepo := &mockValidationRepository{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			return v, true, nil
		},
		findByIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, error) {
			return v, nil
		},
	}
	uc := newGetLatestProblemValidationUseCase(repo, validationRepo, &mockProblemStatusProvider{})

	out, err := uc.Execute(context.Background(), GetLatestProblemValidationInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Found {
		t.Error("expected Found=true — there is a ticket, it's just not finished")
	}
	if out.Terminal {
		t.Error("expected Terminal=false while the ticket is still RUNNING")
	}
}
