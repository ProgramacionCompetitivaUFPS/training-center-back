package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestUpdatePassword_Success(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == "user-1" {
			return activeUser, nil
		}
		return nil, nil
	}
	uc := NewUpdatePasswordUseCase(repo, &mockEmailSender{}, &mockSessionInvalidator{}, &mockRateLimiter{})

	_, err := uc.Execute(context.Background(), UpdatePasswordInput{
		UserID:          "user-1",
		CurrentPassword: "Secret1!",
		NewPassword:     "NewSecret2@",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if activeUser.UpdatedAt() == nil {
		t.Error("expected updatedAt to be set after password change")
	}
	if activeUser.Password().Compare("Secret1!") {
		t.Error("expected old password to be invalid after update")
	}
	if !activeUser.Password().Compare("NewSecret2@") {
		t.Error("expected new password to be valid after update")
	}
}

func TestUpdatePassword_UserNotFound(t *testing.T) {
	repo := newNoConflictRepo()
	uc := NewUpdatePasswordUseCase(repo, &mockEmailSender{}, &mockSessionInvalidator{}, &mockRateLimiter{})

	_, err := uc.Execute(context.Background(), UpdatePasswordInput{
		UserID:          "nonexistent",
		CurrentPassword: "Secret1!",
		NewPassword:     "NewSecret2@",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != domain.ErrCodeUserNotFound {
		t.Errorf("expected code NOT_FOUND, got %q", appErr.Code)
	}
	if appErr.Kind != apperror.KindNotFound {
		t.Errorf("expected kind NOT_FOUND, got %s", appErr.Kind)
	}
}

func TestUpdatePassword_WrongCurrentPassword(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return activeUser, nil
	}
	uc := NewUpdatePasswordUseCase(repo, &mockEmailSender{}, &mockSessionInvalidator{}, &mockRateLimiter{})

	_, err := uc.Execute(context.Background(), UpdatePasswordInput{
		UserID:          "user-1",
		CurrentPassword: "WrongPassword1!",
		NewPassword:     "NewSecret2@",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected code VALIDATION_ERROR, got %q", appErr.Code)
	}
	if appErr.Kind != apperror.KindValidation {
		t.Errorf("expected kind VALIDATION, got %s", appErr.Kind)
	}
	if len(appErr.Details) == 0 || appErr.Details[0].Message != "Current password is incorrect" {
		t.Errorf("expected message %q, got %v", "Current password is incorrect", appErr.Details)
	}
}

func TestUpdatePassword_WeakNewPassword(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return activeUser, nil
	}
	uc := NewUpdatePasswordUseCase(repo, &mockEmailSender{}, &mockSessionInvalidator{}, &mockRateLimiter{})

	_, err := uc.Execute(context.Background(), UpdatePasswordInput{
		UserID:          "user-1",
		CurrentPassword: "Secret1!",
		NewPassword:     "short",
	})
	if err == nil {
		t.Fatal("expected error for weak password, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected code VALIDATION_ERROR, got %q", appErr.Code)
	}
	if len(appErr.Details) == 0 || appErr.Details[0].Field != "newPassword" {
		t.Errorf("expected field error on newPassword, got %v", appErr.Details)
	}
}

func TestUpdatePassword_SamePassword(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return activeUser, nil
	}
	uc := NewUpdatePasswordUseCase(repo, &mockEmailSender{}, &mockSessionInvalidator{}, &mockRateLimiter{})

	_, err := uc.Execute(context.Background(), UpdatePasswordInput{
		UserID:          "user-1",
		CurrentPassword: "Secret1!",
		NewPassword:     "Secret1!",
	})
	if err == nil {
		t.Fatal("expected error when new password equals current, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected code VALIDATION_ERROR, got %q", appErr.Code)
	}
	if len(appErr.Details) == 0 || appErr.Details[0].Message != "New password must be different from current password" {
		t.Errorf("expected 'must be different' message, got %v", appErr.Details)
	}
}

func TestUpdatePassword_RepositoryFindError(t *testing.T) {
	repo := newNoConflictRepo()
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return nil, apperror.NewInternal()
	}
	uc := NewUpdatePasswordUseCase(repo, &mockEmailSender{}, &mockSessionInvalidator{}, &mockRateLimiter{})

	_, err := uc.Execute(context.Background(), UpdatePasswordInput{
		UserID:          "user-1",
		CurrentPassword: "Secret1!",
		NewPassword:     "NewSecret2@",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected code INTERNAL_ERROR, got %q", appErr.Code)
	}
}

func TestUpdatePassword_RepositoryUpdateError(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return activeUser, nil
	}
	repo.updateFn = func(_ context.Context, _ *domain.User) error {
		return apperror.NewInternal()
	}
	uc := NewUpdatePasswordUseCase(repo, &mockEmailSender{}, &mockSessionInvalidator{}, &mockRateLimiter{})

	_, err := uc.Execute(context.Background(), UpdatePasswordInput{
		UserID:          "user-1",
		CurrentPassword: "Secret1!",
		NewPassword:     "NewSecret2@",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected code INTERNAL_ERROR, got %q", appErr.Code)
	}
}

func TestUpdatePassword_SessionInvalidationFails_PasswordAlreadyChanged(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return activeUser, nil
	}
	inv := &mockSessionInvalidator{
		invalidateAllUserSessionsFn: func(_ context.Context, _ string, _ time.Time) error {
			return errors.New("redis unavailable")
		},
	}
	uc := NewUpdatePasswordUseCase(repo, &mockEmailSender{}, inv, &mockRateLimiter{})

	out, err := uc.Execute(context.Background(), UpdatePasswordInput{
		UserID:          "user-1",
		CurrentPassword: "Secret1!",
		NewPassword:     "NewSecret2@",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.SessionsInvalidated {
		t.Fatal("expected sessions not to be invalidated")
	}

	// Password must have been persisted despite the session invalidation failure.
	if !activeUser.Password().Compare("NewSecret2@") {
		t.Error("expected new password to be valid: repo.Update must run before session invalidation")
	}
}
