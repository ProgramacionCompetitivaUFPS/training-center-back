package user

import (
	"context"
	"errors"
	"testing"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestRequestEmailChange_Success(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == "user-1" {
			return activeUser, nil
		}
		return nil, nil
	}
	emailChangeRepo := &mockEmailChangeRepo{}
	
	emailSent := false
	mockEmail := &mockEmailSender{
		sendFn: func(ctx context.Context, msg appshared.EmailMessage) error {
			emailSent = true
			if msg.To != "newemail@example.com" {
				t.Errorf("expected email to be sent to newemail@example.com, got %s", msg.To)
			}
			return nil
		},
	}

	uc := NewRequestEmailChangeUseCase(userRepo, emailChangeRepo, mockEmail, &mockRateLimiter{})

	_, err := uc.Execute(context.Background(), RequestEmailChangeInput{
		UserID:   "user-1",
		Password: "Secret1!",
		NewEmail: "newemail@example.com",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !emailSent {
		t.Error("expected verification email to be sent")
	}
}

func TestRequestEmailChange_WrongPassword(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == "user-1" {
			return activeUser, nil
		}
		return nil, nil
	}

	emailChangeRepo := &mockEmailChangeRepo{}
	mockEmail := &mockEmailSender{}
	uc := NewRequestEmailChangeUseCase(userRepo, emailChangeRepo, mockEmail, &mockRateLimiter{})

	_, err := uc.Execute(context.Background(), RequestEmailChangeInput{
		UserID:   "user-1",
		Password: "WrongPassword!",
		NewEmail: "newemail@example.com",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != ErrCodeInvalidCredentials {
		t.Errorf("expected INVALID_CREDENTIALS error, got %v", err)
	}
}

func TestRequestEmailChange_UserNotFound(t *testing.T) {
	userRepo := newNoConflictRepo()
	userRepo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return nil, nil
	}

	uc := NewRequestEmailChangeUseCase(userRepo, &mockEmailChangeRepo{}, &mockEmailSender{}, &mockRateLimiter{})
	_, err := uc.Execute(context.Background(), RequestEmailChangeInput{
		UserID:   "ghost",
		Password: "Secret1!",
		NewEmail: "new@example.com",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != ErrCodeInvalidCredentials {
		t.Errorf("expected INVALID_CREDENTIALS, got %v", err)
	}
}

func TestRequestEmailChange_InvalidNewEmail(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) { return activeUser, nil }

	uc := NewRequestEmailChangeUseCase(userRepo, &mockEmailChangeRepo{}, &mockEmailSender{}, &mockRateLimiter{})
	_, err := uc.Execute(context.Background(), RequestEmailChangeInput{
		UserID:   "user-1",
		Password: "Secret1!",
		NewEmail: "not-an-email",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != "VALIDATION_ERROR" {
		t.Errorf("expected VALIDATION_ERROR, got %v", err)
	}
	if len(appErr.Details) == 0 || appErr.Details[0].Field != "newEmail" {
		t.Errorf("expected field error on newEmail, got %v", appErr.Details)
	}
}

func TestRequestEmailChange_EmailDeliveryFails(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == "user-1" {
			return activeUser, nil
		}
		return nil, nil
	}
	emailChangeRepo := &mockEmailChangeRepo{}
	
	// Simulate SMTP failure
	mockEmail := &mockEmailSender{
		sendFn: func(ctx context.Context, msg appshared.EmailMessage) error {
			return errors.New("smtp timeout")
		},
	}

	uc := NewRequestEmailChangeUseCase(userRepo, emailChangeRepo, mockEmail, &mockRateLimiter{})

	_, err := uc.Execute(context.Background(), RequestEmailChangeInput{
		UserID:   "user-1",
		Password: "Secret1!",
		NewEmail: "newemail@example.com",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != ErrCodeEmailDeliveryFailed {
		t.Errorf("expected EMAIL_DELIVERY_FAILED error, got %v", err)
	}
	if appErr.Kind != apperror.KindServiceUnavailable {
		t.Errorf("expected kind SERVICE_UNAVAILABLE, got %s", appErr.Kind)
	}
}
