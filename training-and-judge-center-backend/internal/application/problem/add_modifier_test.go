package problem

import (
	"context"
	"errors"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const newModifierID = "cccccccc-0000-0000-0000-000000000001"

func TestAddModifier_Success_Author(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := NewAddModifierUseCase(repo, &mockUserProvider{})

	err := uc.Execute(context.Background(), AddModifierInput{
		Slug:        testSlug,
		UserID:      newModifierID,
		CurrentUser: asCoach(authorID),
	})
	if err != nil {
		t.Fatalf("author should be able to add a modifier, got: %v", err)
	}
}

func TestAddModifier_Success_Admin(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := NewAddModifierUseCase(repo, &mockUserProvider{})

	err := uc.Execute(context.Background(), AddModifierInput{
		Slug:        testSlug,
		UserID:      newModifierID,
		CurrentUser: asAdmin(strangerID),
	})
	if err != nil {
		t.Fatalf("admin should be able to add a modifier, got: %v", err)
	}
}

func TestAddModifier_Forbidden_Stranger(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := NewAddModifierUseCase(repo, &mockUserProvider{})

	err := uc.Execute(context.Background(), AddModifierInput{
		Slug:        testSlug,
		UserID:      newModifierID,
		CurrentUser: asContestant(strangerID),
	})
	if err == nil {
		t.Fatal("stranger should not add modifiers, got nil error")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeForbidden {
		t.Errorf("expected FORBIDDEN, got %q", appErr.Code)
	}
}

func TestAddModifier_UserNotFound(t *testing.T) {
	repo := repoWith(newDraftProblem())
	provider := &mockUserProvider{
		existsByIDFn: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
	}
	uc := NewAddModifierUseCase(repo, provider)

	err := uc.Execute(context.Background(), AddModifierInput{
		Slug:        testSlug,
		UserID:      "nonexistent-user-id-x",
		CurrentUser: asCoach(authorID),
	})
	if err == nil {
		t.Fatal("expected not-found error for unknown user, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != "USER_NOT_FOUND" {
		t.Errorf("expected USER_NOT_FOUND, got %q", appErr.Code)
	}
}

func TestAddModifier_UserAlreadyModifier(t *testing.T) {
	repo := repoWith(newDraftProblemWithModifier()) // modifierID already in list
	uc := NewAddModifierUseCase(repo, &mockUserProvider{})

	err := uc.Execute(context.Background(), AddModifierInput{
		Slug:        testSlug,
		UserID:      modifierID, // already a modifier
		CurrentUser: asCoach(authorID),
	})
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != domainProblem.ErrCodeModifierAlreadyExists {
		t.Errorf("expected %q, got %q", domainProblem.ErrCodeModifierAlreadyExists, appErr.Code)
	}
}

func TestAddModifier_UserProviderError(t *testing.T) {
	repo := repoWith(newDraftProblem())
	provider := &mockUserProvider{
		existsByIDFn: func(_ context.Context, _ string) (bool, error) {
			return false, errors.New("provider down")
		},
	}
	uc := NewAddModifierUseCase(repo, provider)

	err := uc.Execute(context.Background(), AddModifierInput{
		Slug:        testSlug,
		UserID:      newModifierID,
		CurrentUser: asCoach(authorID),
	})
	if err == nil {
		t.Fatal("expected internal error when provider fails, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected INTERNAL_ERROR, got %q", appErr.Code)
	}
}

func TestAddModifier_InvalidUserID(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := NewAddModifierUseCase(repo, &mockUserProvider{})

	err := uc.Execute(context.Background(), AddModifierInput{
		Slug:        testSlug,
		UserID:      "", // empty ID should fail NewUserID validation
		CurrentUser: asCoach(authorID),
	})
	if err == nil {
		t.Fatal("expected error for empty user ID, got nil")
	}
}
