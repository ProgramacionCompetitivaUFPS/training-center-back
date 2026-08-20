package user

import (
	"context"
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newGoogleOnlyUser(id string, status domain.Status) *domain.User {
	emailStr := id + "@example.com"
	return domain.RestoreUser(
		id,
		&emailStr,
		"",
		"User "+id,
		id,
		"",
		"",
		"",
		shared.RoleContestant.String(),
		status.String(),
		time.Now(),
		nil,
		nil,
	)
}

func TestSetPassword_Success(t *testing.T) {
	googleOnlyUser := newGoogleOnlyUser("user-1", domain.StatusActive)
	repo := newNoConflictRepo()
	repo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == "user-1" {
			return googleOnlyUser, nil
		}
		return nil, nil
	}
	var savedUser *domain.User
	repo.updateFn = func(_ context.Context, u *domain.User) error {
		savedUser = u
		return nil
	}
	uc := NewSetPasswordUseCase(repo, &mockEmailSender{})

	err := uc.Execute(context.Background(), SetPasswordInput{UserID: "user-1", NewPassword: "NewSecret1!"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if savedUser == nil || !savedUser.Password().HasPassword() {
		t.Fatal("expected user to be saved with a password set")
	}
	if !savedUser.Password().Compare("NewSecret1!") {
		t.Error("expected saved password to match the new password")
	}
}

func TestSetPassword_AlreadyHasPassword_ReturnsConflict(t *testing.T) {
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	repo := newNoConflictRepo()
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return activeUser, nil
	}
	uc := NewSetPasswordUseCase(repo, &mockEmailSender{})

	err := uc.Execute(context.Background(), SetPasswordInput{UserID: "user-1", NewPassword: "NewSecret1!"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != ErrCodePasswordAlreadySet {
		t.Errorf("expected code %q, got %q", ErrCodePasswordAlreadySet, appErr.Code)
	}
	if appErr.Kind != apperror.KindConflict {
		t.Errorf("expected kind CONFLICT, got %s", appErr.Kind)
	}
}

func TestSetPassword_InvalidPassword_ReturnsValidation(t *testing.T) {
	googleOnlyUser := newGoogleOnlyUser("user-1", domain.StatusActive)
	repo := newNoConflictRepo()
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return googleOnlyUser, nil
	}
	uc := NewSetPasswordUseCase(repo, &mockEmailSender{})

	err := uc.Execute(context.Background(), SetPasswordInput{UserID: "user-1", NewPassword: "short"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Kind != apperror.KindValidation {
		t.Errorf("expected kind VALIDATION, got %s", appErr.Kind)
	}
}

func TestSetPassword_UserNotFound_ReturnsNotFound(t *testing.T) {
	repo := newNoConflictRepo()
	uc := NewSetPasswordUseCase(repo, &mockEmailSender{})

	err := uc.Execute(context.Background(), SetPasswordInput{UserID: "nonexistent", NewPassword: "NewSecret1!"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != domain.ErrCodeUserNotFound {
		t.Errorf("expected code %q, got %q", domain.ErrCodeUserNotFound, appErr.Code)
	}
}

func TestSetPassword_DeactivatedUser_ReturnsInternal(t *testing.T) {
	deactivatedUser := newGoogleOnlyUser("user-1", domain.StatusDeactivated)
	repo := newNoConflictRepo()
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return deactivatedUser, nil
	}
	uc := NewSetPasswordUseCase(repo, &mockEmailSender{})

	err := uc.Execute(context.Background(), SetPasswordInput{UserID: "user-1", NewPassword: "NewSecret1!"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected code %q, got %q", apperror.ErrCodeInternalError, appErr.Code)
	}
}
