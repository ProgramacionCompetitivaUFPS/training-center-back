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

func TestResetPassword_Success(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	userRepo.findByEmailFn = func(_ context.Context, email domain.Email) (*domain.User, error) {
		return activeUser, nil
	}

	recoveryRepo := &mockPasswordRecoveryRepo{
		findPendingByUserIDFn: func(ctx context.Context, userID string) (*domain.PasswordRecoveryRequest, error) {
			return domain.RestorePasswordRecoveryRequest("req-1", "user-1", "123456", domain.StatusPending, time.Now().Add(10 * time.Minute), time.Time{}, nil), nil
		},
		updateFn: func(ctx context.Context, req *domain.PasswordRecoveryRequest) error {
			if req.Status() != domain.StatusUsed {
				t.Errorf("expected status to be USED")
			}
			return nil
		},
	}

	invCalled := false
	invalidator := &mockSessionInvalidator{
		invalidateAllUserSessionsFn: func(ctx context.Context, userID string, timestamp time.Time) error {
			invCalled = true
			if userID != "user-1" {
				t.Errorf("expected user-1, got %s", userID)
			}
			return nil
		},
	}

	uc := NewResetPasswordUseCase(userRepo, recoveryRepo, invalidator, &mockTransactionManager{})

	err := uc.Execute(context.Background(), ResetPasswordInput{
		Email:       "user-1@example.com",
		Code:        "123456",
		NewPassword: "NewSecret123!",
	})

	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if !invCalled {
		t.Error("expected session invalidator to be called")
	}
}

func TestResetPassword_InvalidCode(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	userRepo.findByEmailFn = func(_ context.Context, email domain.Email) (*domain.User, error) {
		return activeUser, nil
	}

	recoveryRepo := &mockPasswordRecoveryRepo{
		findPendingByUserIDFn: func(ctx context.Context, userID string) (*domain.PasswordRecoveryRequest, error) {
			return domain.RestorePasswordRecoveryRequest("req-1", "user-1", "654321", domain.StatusPending, time.Now().Add(10*time.Minute), time.Time{}, nil), nil
		},
	}

	uc := NewResetPasswordUseCase(userRepo, recoveryRepo, &mockSessionInvalidator{}, &mockTransactionManager{})

	err := uc.Execute(context.Background(), ResetPasswordInput{
		Email:       "user-1@example.com",
		Code:        "123456",
		NewPassword: "NewSecret123!",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != "INVALID_RECOVERY_ATTEMPT" {
		t.Errorf("expected INVALID_RECOVERY_ATTEMPT, got %v", err)
	}
}

func TestResetPassword_NoPendingRequest(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	userRepo.findByEmailFn = func(_ context.Context, email domain.Email) (*domain.User, error) {
		return activeUser, nil
	}

	recoveryRepo := &mockPasswordRecoveryRepo{
		findPendingByUserIDFn: func(ctx context.Context, userID string) (*domain.PasswordRecoveryRequest, error) {
			return nil, nil // No pending requests
		},
	}

	uc := NewResetPasswordUseCase(userRepo, recoveryRepo, &mockSessionInvalidator{}, &mockTransactionManager{})
	err := uc.Execute(context.Background(), ResetPasswordInput{
		Email:       "user-1@example.com",
		Code:        "123456",
		NewPassword: "NewSecret123!",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != "INVALID_RECOVERY_ATTEMPT" {
		t.Errorf("expected INVALID_RECOVERY_ATTEMPT, got %v", err)
	}
}

func TestResetPassword_ExpiredRequest(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	userRepo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return activeUser, nil
	}

	recoveryRepo := &mockPasswordRecoveryRepo{
		findPendingByUserIDFn: func(_ context.Context, _ string) (*domain.PasswordRecoveryRequest, error) {
			// Request exists in DB but its window has already elapsed
			return domain.RestorePasswordRecoveryRequest("req-1", "user-1", "123456", domain.StatusPending, time.Now().Add(-1*time.Minute), time.Time{}, nil), nil
		},
	}

	uc := NewResetPasswordUseCase(userRepo, recoveryRepo, &mockSessionInvalidator{}, &mockTransactionManager{})

	err := uc.Execute(context.Background(), ResetPasswordInput{
		Email:       "user-1@example.com",
		Code:        "123456",
		NewPassword: "NewSecret123!",
	})

	if err == nil {
		t.Fatal("expected error for expired request, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != "INVALID_RECOVERY_ATTEMPT" {
		t.Errorf("expected INVALID_RECOVERY_ATTEMPT, got %v", err)
	}
}

func TestResetPassword_WeakPassword(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	userRepo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return activeUser, nil
	}

	recoveryRepo := &mockPasswordRecoveryRepo{
		findPendingByUserIDFn: func(_ context.Context, _ string) (*domain.PasswordRecoveryRequest, error) {
			return domain.RestorePasswordRecoveryRequest("req-1", "user-1", "123456", domain.StatusPending, time.Now().Add(10*time.Minute), time.Time{}, nil), nil
		},
	}

	txCalled := false
	invCalled := false
	uc := NewResetPasswordUseCase(
		userRepo,
		recoveryRepo,
		&mockSessionInvalidator{
			invalidateAllUserSessionsFn: func(_ context.Context, _ string, _ time.Time) error {
				invCalled = true
				return nil
			},
		},
		&mockTransactionManager{
			withTxFn: func(_ context.Context, _ func(context.Context) error) error {
				txCalled = true
				return nil
			},
		},
	)

	err := uc.Execute(context.Background(), ResetPasswordInput{
		Email:       "user-1@example.com",
		Code:        "123456",
		NewPassword: "weak",
	})

	if err == nil {
		t.Fatal("expected validation error for weak password, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != "VALIDATION_ERROR" {
		t.Errorf("expected VALIDATION_ERROR, got %v", err)
	}
	if txCalled {
		t.Error("transaction must NOT be started when password validation fails")
	}
	if invCalled {
		t.Error("session invalidator must NOT be called when password validation fails")
	}
}

func TestResetPassword_UserNotFound(t *testing.T) {
	userRepo := newNoConflictRepo()
	userRepo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return nil, nil // email does not exist in DB
	}

	uc := NewResetPasswordUseCase(userRepo, &mockPasswordRecoveryRepo{}, &mockSessionInvalidator{}, &mockTransactionManager{})

	err := uc.Execute(context.Background(), ResetPasswordInput{
		Email:       "ghost@example.com",
		Code:        "123456",
		NewPassword: "NewSecret123!",
	})

	if err == nil {
		t.Fatal("expected error when user not found, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != "INVALID_RECOVERY_ATTEMPT" {
		t.Errorf("expected INVALID_RECOVERY_ATTEMPT to avoid information leakage, got %v", err)
	}
}

func TestResetPassword_TxFailure(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	userRepo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return activeUser, nil
	}

	recoveryRepo := &mockPasswordRecoveryRepo{
		findPendingByUserIDFn: func(_ context.Context, _ string) (*domain.PasswordRecoveryRequest, error) {
			return domain.RestorePasswordRecoveryRequest("req-1", "user-1", "123456", domain.StatusPending, time.Now().Add(10*time.Minute), time.Time{}, nil), nil
		},
	}

	tx := &mockTransactionManager{
		withTxFn: func(_ context.Context, _ func(context.Context) error) error {
			return errors.New("db error")
		},
	}

	uc := NewResetPasswordUseCase(userRepo, recoveryRepo, &mockSessionInvalidator{}, tx)
	err := uc.Execute(context.Background(), ResetPasswordInput{
		Email:       "user-1@example.com",
		Code:        "123456",
		NewPassword: "NewSecret123!",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != "INTERNAL_ERROR" {
		t.Errorf("expected INTERNAL_ERROR, got %v", err)
	}
}

func TestResetPassword_SessionInvalidationFailure(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	userRepo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return activeUser, nil
	}

	recoveryRepo := &mockPasswordRecoveryRepo{
		findPendingByUserIDFn: func(_ context.Context, _ string) (*domain.PasswordRecoveryRequest, error) {
			return domain.RestorePasswordRecoveryRequest("req-1", "user-1", "123456", domain.StatusPending, time.Now().Add(10*time.Minute), time.Time{}, nil), nil
		},
		updateFn: func(_ context.Context, _ *domain.PasswordRecoveryRequest) error { return nil },
	}

	invalidator := &mockSessionInvalidator{
		invalidateAllUserSessionsFn: func(_ context.Context, _ string, _ time.Time) error {
			return errors.New("redis unavailable")
		},
	}

	uc := NewResetPasswordUseCase(userRepo, recoveryRepo, invalidator, &mockTransactionManager{})
	err := uc.Execute(context.Background(), ResetPasswordInput{
		Email:       "user-1@example.com",
		Code:        "123456",
		NewPassword: "NewSecret123!",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != ErrSessionsNotInvalidated {
		t.Errorf("expected ErrSessionsNotInvalidated, got %v", err)
	}
}
