package user

import (
	"context"
	"testing"
	"time"

	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestResetPassword_Success(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
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
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
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
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
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
