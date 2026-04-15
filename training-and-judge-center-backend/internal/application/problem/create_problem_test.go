package problem

import (
	"context"
	"errors"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func validCreateInput() CreateProblemInput {
	return CreateProblemInput{
		Slug:        testSlug,
		Title:       "Test Problem",
		CurrentUser: asCoach(authorID),
	}
}

func TestCreateProblem_Success_Coach(t *testing.T) {
	repo := &mockProblemRepository{}
	uc := NewCreateProblemUseCase(repo, newDefaultSettings())

	result, err := uc.Execute(context.Background(), validCreateInput())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Problem.Slug().String() != testSlug {
		t.Errorf("expected slug %q, got %q", testSlug, result.Problem.Slug().String())
	}
	if result.Problem.Status().String() != "DRAFT" {
		t.Errorf("expected DRAFT status, got %q", result.Problem.Status().String())
	}
	if result.Problem.AuthorID().Value() != authorID {
		t.Errorf("expected authorID %q, got %q", authorID, result.Problem.AuthorID().Value())
	}
}

func TestCreateProblem_Success_Admin(t *testing.T) {
	repo := &mockProblemRepository{}
	uc := NewCreateProblemUseCase(repo, newDefaultSettings())

	input := validCreateInput()
	input.CurrentUser = asAdmin(authorID)

	_, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("admin should be able to create a problem, got error: %v", err)
	}
}

func TestCreateProblem_Forbidden_Contestant(t *testing.T) {
	repo := &mockProblemRepository{}
	uc := NewCreateProblemUseCase(repo, newDefaultSettings())

	input := validCreateInput()
	input.CurrentUser = asContestant(strangerID)

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeForbidden {
		t.Errorf("expected FORBIDDEN, got %q", appErr.Code)
	}
}

func TestCreateProblem_InvalidSlug(t *testing.T) {
	repo := &mockProblemRepository{}
	uc := NewCreateProblemUseCase(repo, newDefaultSettings())

	input := validCreateInput()
	input.Slug = "ab" // too short (min 3 chars)

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %q", appErr.Code)
	}
}

func TestCreateProblem_InvalidTitle(t *testing.T) {
	repo := &mockProblemRepository{}
	uc := NewCreateProblemUseCase(repo, newDefaultSettings())

	input := validCreateInput()
	input.Title = "" // empty title

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %q", appErr.Code)
	}
}

func TestCreateProblem_InvalidTag(t *testing.T) {
	repo := &mockProblemRepository{}
	uc := NewCreateProblemUseCase(repo, newDefaultSettings())

	input := validCreateInput()
	input.Tags = []string{"not-a-valid-tag"}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected validation error for unknown tag, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %q", appErr.Code)
	}
}

func TestCreateProblem_SlugAlreadyExists(t *testing.T) {
	repo := &mockProblemRepository{
		saveFn: func(_ context.Context, p *domainProblem.Problem) error {
			return &domainProblem.ErrSlugAlreadyExists{Slug: p.Slug().String()}
		},
	}
	uc := NewCreateProblemUseCase(repo, newDefaultSettings())

	_, err := uc.Execute(context.Background(), validCreateInput())
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != ErrCodeProblemSlugAlreadyExists {
		t.Errorf("expected %q, got %q", ErrCodeProblemSlugAlreadyExists, appErr.Code)
	}
}

func TestCreateProblem_RepositoryError(t *testing.T) {
	repo := &mockProblemRepository{
		saveFn: func(_ context.Context, _ *domainProblem.Problem) error {
			return errors.New("db connection lost")
		},
	}
	uc := NewCreateProblemUseCase(repo, newDefaultSettings())

	_, err := uc.Execute(context.Background(), validCreateInput())
	if err == nil {
		t.Fatal("expected internal error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected INTERNAL_ERROR, got %q", appErr.Code)
	}
}
