package user

import (
	"context"
	"errors"
	"net/http"
	"testing"

	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type mockEmailSender struct {
	sendFn func(ctx context.Context, to, subject, body string) error
}

func (m *mockEmailSender) Send(ctx context.Context, to, subject, body string) error {
	if m.sendFn != nil {
		return m.sendFn(ctx, to, subject, body)
	}
	return nil
}

func TestUpdatePassword_Success(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == "user-1" {
			return activeUser, nil
		}
		return nil, nil
	}
	mockEmail := &mockEmailSender{}
	uc := NewUpdatePasswordUseCase(repo, mockEmail)

	err := uc.Execute(context.Background(), "user-1", UpdatePasswordInput{
		CurrentPassword: "Secret1!",
		NewPassword:     "NewSecret2@",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if activeUser.UpdatedAt == nil {
		t.Error("expected updatedAt to be set after password change")
	}
	if activeUser.Password.Compare("Secret1!") {
		t.Error("expected old password to be invalid after update")
	}
	if !activeUser.Password.Compare("NewSecret2@") {
		t.Error("expected new password to be valid after update")
	}
}

func TestUpdatePassword_UserNotFound(t *testing.T) {
	repo := newNoConflictRepo()
	mockEmail := &mockEmailSender{}
	uc := NewUpdatePasswordUseCase(repo, mockEmail)

	err := uc.Execute(context.Background(), "nonexistent", UpdatePasswordInput{
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
	if appErr.Code != "NOT_FOUND" {
		t.Errorf("expected code NOT_FOUND, got %q", appErr.Code)
	}
	if appErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", appErr.StatusCode)
	}
}

func TestUpdatePassword_WrongCurrentPassword(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return activeUser, nil
	}
	mockEmail := &mockEmailSender{}
	uc := NewUpdatePasswordUseCase(repo, mockEmail)

	err := uc.Execute(context.Background(), "user-1", UpdatePasswordInput{
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
	if appErr.Code != "VALIDATION_ERROR" {
		t.Errorf("expected code VALIDATION_ERROR, got %q", appErr.Code)
	}
	if appErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", appErr.StatusCode)
	}
	if len(appErr.Details) == 0 || appErr.Details[0].Message != "Current password is incorrect" {
		t.Errorf("expected message %q, got %v", "Current password is incorrect", appErr.Details)
	}
}

func TestUpdatePassword_WeakNewPassword(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return activeUser, nil
	}
	mockEmail := &mockEmailSender{}
	uc := NewUpdatePasswordUseCase(repo, mockEmail)

	err := uc.Execute(context.Background(), "user-1", UpdatePasswordInput{
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
	if appErr.Code != "VALIDATION_ERROR" {
		t.Errorf("expected code VALIDATION_ERROR, got %q", appErr.Code)
	}
	if len(appErr.Details) == 0 || appErr.Details[0].Field != "newPassword" {
		t.Errorf("expected field error on newPassword, got %v", appErr.Details)
	}
}

func TestUpdatePassword_SamePassword(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return activeUser, nil
	}
	mockEmail := &mockEmailSender{}
	uc := NewUpdatePasswordUseCase(repo, mockEmail)

	err := uc.Execute(context.Background(), "user-1", UpdatePasswordInput{
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
	if appErr.Code != "VALIDATION_ERROR" {
		t.Errorf("expected code VALIDATION_ERROR, got %q", appErr.Code)
	}
	if len(appErr.Details) == 0 || appErr.Details[0].Message != "New password must be different from current password" {
		t.Errorf("expected 'must be different' message, got %v", appErr.Details)
	}
}

func TestUpdatePassword_RepositoryFindError(t *testing.T) {
	repo := newNoConflictRepo()
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return nil, errors.New("db connection lost")
	}
	mockEmail := &mockEmailSender{}
	uc := NewUpdatePasswordUseCase(repo, mockEmail)

	err := uc.Execute(context.Background(), "user-1", UpdatePasswordInput{
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
	if appErr.Code != "INTERNAL_ERROR" {
		t.Errorf("expected code INTERNAL_ERROR, got %q", appErr.Code)
	}
}

func TestUpdatePassword_RepositoryUpdateError(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return activeUser, nil
	}
	repo.updateFn = func(_ context.Context, _ *domain.User) error {
		return errors.New("db update failed")
	}
	mockEmail := &mockEmailSender{}
	uc := NewUpdatePasswordUseCase(repo, mockEmail)

	err := uc.Execute(context.Background(), "user-1", UpdatePasswordInput{
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
	if appErr.Code != "INTERNAL_ERROR" {
		t.Errorf("expected code INTERNAL_ERROR, got %q", appErr.Code)
	}
}
