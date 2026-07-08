package problem

import (
	"context"
	"testing"

	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestListModifiers_Forbidden_Stranger(t *testing.T) {
	repo := repoWith(newDraftProblemWithModifier())
	uc := NewListModifiersUseCase(repo, &mockUserProvider{})

	_, err := uc.Execute(context.Background(), ListModifiersInput{
		Slug:        testSlug,
		CurrentUser: asContestant(strangerID),
	})
	if err == nil {
		t.Fatal("stranger should not list modifiers, got nil error")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != ErrCodeInsufficientPermissions {
		t.Errorf("expected INSUFFICIENT_PERMISSIONS, got %q", appErr.Code)
	}
}

func TestListModifiers_Success_ResolvesNicknameAndName(t *testing.T) {
	repo := repoWith(newDraftProblemWithModifier())
	provider := &mockUserProvider{
		getDisplaysFn: func(_ context.Context, userIDs []string) (map[string]*UserDisplay, error) {
			out := make(map[string]*UserDisplay, len(userIDs))
			for _, id := range userIDs {
				out[id] = &UserDisplay{Nickname: "coach_jane", Name: "Jane Doe"}
			}
			return out, nil
		},
	}
	uc := NewListModifiersUseCase(repo, provider)

	out, err := uc.Execute(context.Background(), ListModifiersInput{
		Slug:        testSlug,
		CurrentUser: asCoach(authorID),
	})
	if err != nil {
		t.Fatalf("author should list modifiers, got: %v", err)
	}
	if len(out.Modifiers) != 1 {
		t.Fatalf("expected 1 modifier, got %d", len(out.Modifiers))
	}
	if out.Modifiers[0].Nickname != "coach_jane" || out.Modifiers[0].Name != "Jane Doe" {
		t.Errorf("expected {coach_jane, Jane Doe}, got %+v", out.Modifiers[0])
	}
}

func TestListModifiers_Success_NoModifiers_ReturnsEmptyList(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := NewListModifiersUseCase(repo, &mockUserProvider{})

	out, err := uc.Execute(context.Background(), ListModifiersInput{
		Slug:        testSlug,
		CurrentUser: asCoach(authorID),
	})
	if err != nil {
		t.Fatalf("author should list modifiers, got: %v", err)
	}
	if len(out.Modifiers) != 0 {
		t.Errorf("expected 0 modifiers, got %d", len(out.Modifiers))
	}
}

func TestListModifiers_UserProviderError(t *testing.T) {
	repo := repoWith(newDraftProblemWithModifier())
	provider := &mockUserProvider{
		getDisplaysFn: func(_ context.Context, _ []string) (map[string]*UserDisplay, error) {
			return nil, apperror.NewInternal()
		},
	}
	uc := NewListModifiersUseCase(repo, provider)

	_, err := uc.Execute(context.Background(), ListModifiersInput{
		Slug:        testSlug,
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
