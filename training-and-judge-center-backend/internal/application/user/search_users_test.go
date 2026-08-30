package user

import (
	"context"
	"errors"
	"testing"

	"github.com/training-judge-center/backend/internal/domain/shared"
	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestSearchUsers_Success(t *testing.T) {
	users := []*domain.User{
		newUserWithRole("user-1", shared.RoleCoach, domain.StatusActive),
	}
	var capturedTerm string
	var capturedLimit int
	repo := &mockUserRepository{
		searchActiveFn: func(_ context.Context, term string, limit int) ([]*domain.User, error) {
			capturedTerm = term
			capturedLimit = limit
			return users, nil
		},
	}
	uc := NewSearchUsersUseCase(repo)

	out, err := uc.Execute(context.Background(), SearchUsersInput{Query: "user-1"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Users) != 1 {
		t.Fatalf("expected 1 result, got %d", len(out.Users))
	}
	if out.Users[0].Nickname != "user-1" {
		t.Errorf("expected nickname %q, got %q", "user-1", out.Users[0].Nickname)
	}
	if capturedTerm != "user-1" {
		t.Errorf("expected term %q, got %q", "user-1", capturedTerm)
	}
	if capturedLimit != defaultSearchLimit {
		t.Errorf("expected default limit %d, got %d", defaultSearchLimit, capturedLimit)
	}
}

func TestSearchUsers_QueryTooShortReturnsValidationError(t *testing.T) {
	uc := NewSearchUsersUseCase(&mockUserRepository{})

	_, err := uc.Execute(context.Background(), SearchUsersInput{Query: "a"})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindValidation {
		t.Errorf("expected validation error, got %v", err)
	}
}

func TestSearchUsers_LimitClampedToMax(t *testing.T) {
	var capturedLimit int
	repo := &mockUserRepository{
		searchActiveFn: func(_ context.Context, _ string, limit int) ([]*domain.User, error) {
			capturedLimit = limit
			return nil, nil
		},
	}
	uc := NewSearchUsersUseCase(repo)

	_, err := uc.Execute(context.Background(), SearchUsersInput{Query: "test", Limit: 500})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedLimit != maxSearchLimit {
		t.Errorf("expected limit clamped to %d, got %d", maxSearchLimit, capturedLimit)
	}
}
