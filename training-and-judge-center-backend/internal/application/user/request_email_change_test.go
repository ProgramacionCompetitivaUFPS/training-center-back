package user

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/training-judge-center/backend/internal/domain/notification"
	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestRequestEmailChange_Success(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == "user-1" {
			return activeUser, nil
		}
		return nil, nil
	}
	userRepo.existsByEmailFn = func(_ context.Context, _ domain.Email) (bool, error) {
		return false, nil // Email is unique
	}

	emailChangeRepo := &mockEmailChangeRepo{}
	
	emailSent := false
	mockEmail := &mockEmailSender{
		sendFn: func(ctx context.Context, msg notification.EmailMessage) error {
			emailSent = true
			if msg.To != "newemail@example.com" {
				t.Errorf("expected email to be sent to newemail@example.com, got %s", msg.To)
			}
			return nil
		},
	}

	uc := NewRequestEmailChangeUseCase(userRepo, emailChangeRepo, mockEmail)

	err := uc.Execute(context.Background(), RequestEmailChangeInput{
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
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == "user-1" {
			return activeUser, nil
		}
		return nil, nil
	}

	emailChangeRepo := &mockEmailChangeRepo{}
	mockEmail := &mockEmailSender{}
	uc := NewRequestEmailChangeUseCase(userRepo, emailChangeRepo, mockEmail)

	err := uc.Execute(context.Background(), RequestEmailChangeInput{
		UserID:   "user-1",
		Password: "WrongPassword!",
		NewEmail: "newemail@example.com",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != "INVALID_CREDENTIALS" {
		t.Errorf("expected INVALID_CREDENTIALS error, got %v", err)
	}
}

func TestRequestEmailChange_EmailAlreadyExists(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == "user-1" {
			return activeUser, nil
		}
		return nil, nil
	}
	userRepo.existsByEmailFn = func(_ context.Context, _ domain.Email) (bool, error) {
		return true, nil // Email is already in use
	}

	emailChangeRepo := &mockEmailChangeRepo{}
	mockEmail := &mockEmailSender{}
	uc := NewRequestEmailChangeUseCase(userRepo, emailChangeRepo, mockEmail)

	err := uc.Execute(context.Background(), RequestEmailChangeInput{
		UserID:   "user-1",
		Password: "Secret1!",
		NewEmail: "used@example.com",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != "EMAIL_ALREADY_EXISTS" {
		t.Errorf("expected EMAIL_ALREADY_EXISTS error, got %v", err)
	}
}

func TestRequestEmailChange_EmailDeliveryFails(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == "user-1" {
			return activeUser, nil
		}
		return nil, nil
	}
	userRepo.existsByEmailFn = func(_ context.Context, _ domain.Email) (bool, error) {
		return false, nil
	}

	emailChangeRepo := &mockEmailChangeRepo{}
	
	// Simulate SMTP failure
	mockEmail := &mockEmailSender{
		sendFn: func(ctx context.Context, msg notification.EmailMessage) error {
			return errors.New("smtp timeout")
		},
	}

	uc := NewRequestEmailChangeUseCase(userRepo, emailChangeRepo, mockEmail)

	err := uc.Execute(context.Background(), RequestEmailChangeInput{
		UserID:   "user-1",
		Password: "Secret1!",
		NewEmail: "newemail@example.com",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != "EMAIL_DELIVERY_FAILED" {
		t.Errorf("expected EMAIL_DELIVERY_FAILED error, got %v", err)
	}
	if appErr.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 status code, got %d", appErr.StatusCode)
	}
}
