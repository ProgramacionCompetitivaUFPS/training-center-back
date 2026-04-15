package problem

import (
	"context"
	"errors"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestRemoveModifier_Success_Author(t *testing.T) {
	repo := repoWith(newDraftProblemWithModifier())
	uc := NewRemoveModifierUseCase(repo)

	_, err := uc.Execute(context.Background(), RemoveModifierInput{
		Slug:        testSlug,
		UserID:      modifierID,
		CurrentUser: asCoach(authorID),
	})
	if err != nil {
		t.Fatalf("author should remove a modifier, got: %v", err)
	}
}

func TestRemoveModifier_Success_Admin(t *testing.T) {
	repo := repoWith(newDraftProblemWithModifier())
	uc := NewRemoveModifierUseCase(repo)

	_, err := uc.Execute(context.Background(), RemoveModifierInput{
		Slug:        testSlug,
		UserID:      modifierID,
		CurrentUser: asAdmin(strangerID),
	})
	if err != nil {
		t.Fatalf("admin should remove any modifier, got: %v", err)
	}
}

func TestRemoveModifier_Forbidden_Stranger(t *testing.T) {
	repo := repoWith(newDraftProblemWithModifier())
	uc := NewRemoveModifierUseCase(repo)

	_, err := uc.Execute(context.Background(), RemoveModifierInput{
		Slug:        testSlug,
		UserID:      modifierID,
		CurrentUser: asContestant(strangerID),
	})
	if err == nil {
		t.Fatal("stranger should not remove modifiers, got nil error")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeForbidden {
		t.Errorf("expected FORBIDDEN, got %q", appErr.Code)
	}
}

func TestRemoveModifier_ModifierNotFound(t *testing.T) {
	repo := repoWith(newDraftProblem()) // no modifiers
	uc := NewRemoveModifierUseCase(repo)

	_, err := uc.Execute(context.Background(), RemoveModifierInput{
		Slug:        testSlug,
		UserID:      modifierID,
		CurrentUser: asCoach(authorID),
	})
	if err == nil {
		t.Fatal("expected not-found error when user is not a modifier, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != domainProblem.ErrCodeModifierNotFound {
		t.Errorf("expected %q, got %q", domainProblem.ErrCodeModifierNotFound, appErr.Code)
	}
}

func TestRemoveModifier_RepositoryError(t *testing.T) {
	repo := &mockProblemRepository{
		findBySlugFn: func(_ context.Context, _ domainProblem.Slug) (*domainProblem.Problem, error) {
			return newDraftProblemWithModifier(), nil
		},
		saveFn: func(_ context.Context, _ *domainProblem.Problem) error {
			return errors.New("db error")
		},
	}
	uc := NewRemoveModifierUseCase(repo)

	_, err := uc.Execute(context.Background(), RemoveModifierInput{
		Slug:        testSlug,
		UserID:      modifierID,
		CurrentUser: asCoach(authorID),
	})
	if err == nil {
		t.Fatal("expected internal error on save failure, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected INTERNAL_ERROR, got %q", appErr.Code)
	}
}
