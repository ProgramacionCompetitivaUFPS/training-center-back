package problem

import (
	"context"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestUpdateProblem_Success_Author(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := NewUpdateProblemUseCase(repo, newDefaultSettings())

	newTitle := "Updated Title"
	result, err := uc.Execute(context.Background(), UpdateProblemInput{
		Slug:        testSlug,
		Title:       &newTitle,
		CurrentUser: asCoach(authorID),
	})
	if err != nil {
		t.Fatalf("author should be able to update own draft, got: %v", err)
	}
	if result.Problem.Title != newTitle {
		t.Errorf("expected title %q, got %q", newTitle, result.Problem.Title)
	}
}

func TestUpdateProblem_Success_Modifier(t *testing.T) {
	repo := repoWith(newDraftProblemWithModifier())
	uc := NewUpdateProblemUseCase(repo, newDefaultSettings())

	newTitle := "Modifier's Title"
	_, err := uc.Execute(context.Background(), UpdateProblemInput{
		Slug:        testSlug,
		Title:       &newTitle,
		CurrentUser: asCoach(modifierID),
	})
	if err != nil {
		t.Fatalf("modifier should be able to update draft, got: %v", err)
	}
}

func TestUpdateProblem_Success_Admin(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := NewUpdateProblemUseCase(repo, newDefaultSettings())

	newTitle := "Admin Title"
	_, err := uc.Execute(context.Background(), UpdateProblemInput{
		Slug:        testSlug,
		Title:       &newTitle,
		CurrentUser: asAdmin(strangerID),
	})
	if err != nil {
		t.Fatalf("admin should be able to update any draft, got: %v", err)
	}
}

func TestUpdateProblem_Forbidden_Stranger(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := NewUpdateProblemUseCase(repo, newDefaultSettings())

	newTitle := "Stolen Title"
	_, err := uc.Execute(context.Background(), UpdateProblemInput{
		Slug:        testSlug,
		Title:       &newTitle,
		CurrentUser: asContestant(strangerID),
	})
	if err == nil {
		t.Fatal("stranger should not update the problem, got nil error")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != ErrCodeInsufficientPermissions {
		t.Errorf("expected INSUFFICIENT_PERMISSIONS, got %q", appErr.Code)
	}
}

func TestUpdateProblem_CannotUpdatePublished(t *testing.T) {
	repo := repoWith(newPublishedProblem())
	uc := NewUpdateProblemUseCase(repo, newDefaultSettings())

	newTitle := "New Title"
	_, err := uc.Execute(context.Background(), UpdateProblemInput{
		Slug:        testSlug,
		Title:       &newTitle,
		CurrentUser: asCoach(authorID),
	})
	if err == nil {
		t.Fatal("should not update a published problem, got nil error")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != ErrCodeProblemIsPublished {
		t.Errorf("expected %q, got %q", ErrCodeProblemIsPublished, appErr.Code)
	}
}

func TestUpdateProblem_InvalidTimeLimit(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := NewUpdateProblemUseCase(repo, newDefaultSettings())

	tooHigh := 999999 // exceeds global max of 10000
	_, err := uc.Execute(context.Background(), UpdateProblemInput{
		Slug:        testSlug,
		TimeLimit:   &tooHigh,
		CurrentUser: asCoach(authorID),
	})
	if err == nil {
		t.Fatal("expected validation error for out-of-range time limit, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %q", appErr.Code)
	}
}

func TestUpdateProblem_NotFound(t *testing.T) {
	repo := &mockProblemRepository{
		findBySlugFn: func(_ context.Context, _ domainProblem.Slug) (*domainProblem.Problem, error) {
			return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "problem not found")
		},
	}
	uc := NewUpdateProblemUseCase(repo, newDefaultSettings())

	newTitle := "Title"
	_, err := uc.Execute(context.Background(), UpdateProblemInput{
		Slug:        testSlug,
		Title:       &newTitle,
		CurrentUser: asCoach(authorID),
	})
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
}

func TestUpdateProblem_RepositoryError(t *testing.T) {
	repo := &mockProblemRepository{
		findBySlugFn: func(_ context.Context, _ domainProblem.Slug) (*domainProblem.Problem, error) {
			return newDraftProblem(), nil
		},
		saveFn: func(_ context.Context, _ *domainProblem.Problem) error {
			return apperror.NewInternal()
		},
	}
	uc := NewUpdateProblemUseCase(repo, newDefaultSettings())

	newTitle := "Title"
	_, err := uc.Execute(context.Background(), UpdateProblemInput{
		Slug:        testSlug,
		Title:       &newTitle,
		CurrentUser: asCoach(authorID),
	})
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
