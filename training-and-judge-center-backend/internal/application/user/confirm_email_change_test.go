package user

import (
	"context"
	"net/http"
	"testing"
	"time"

	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestConfirmEmailChange_Success(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == "user-1" {
			return activeUser, nil
		}
		return nil, nil
	}
	userRepo.existsByEmailFn = func(_ context.Context, _ domain.Email) (bool, error) {
		return false, nil // New email is still unique
	}
	var updatedUser *domain.User
	userRepo.updateFn = func(_ context.Context, u *domain.User) error {
		updatedUser = u
		return nil
	}

	mockEmail, _ := domain.NewEmail("newemail@example.com")
	req := &domain.EmailChangeRequest{
		ID:        "req-1",
		UserID:    "user-1",
		NewEmail:  mockEmail,
		Code:      "123456",
		Status:    domain.StatusPending,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	emailChangeRepo := &mockEmailChangeRepo{
		findByCodeAndUserIDFn: func(ctx context.Context, code string, userID string) (*domain.EmailChangeRequest, error) {
			if code == "123456" && userID == "user-1" {
				return req, nil
			}
			return nil, nil
		},
		updateFn: func(ctx context.Context, r *domain.EmailChangeRequest) error {
			if r.Status != domain.StatusExpired && r.Status != domain.StatusUsed {
				t.Errorf("expected request status to be set to USED, got %s", r.Status)
			}
			return nil
		},
	}

	emailsSent := 0
	mockEmailSender := &mockEmailSender{
		sendFn: func(ctx context.Context, to, subject, body string) error {
			emailsSent++
			return nil
		},
	}

	uc := NewConfirmEmailChangeUseCase(userRepo, emailChangeRepo, mockEmailSender)

	newEmail, err := uc.Execute(context.Background(), ConfirmEmailChangeInput{
		UserID: "user-1",
		Code:   "123456",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if newEmail.String() != "newemail@example.com" {
		t.Errorf("expected returned email to be newemail@example.com, got %s", newEmail.String())
	}
	if emailsSent != 2 {
		t.Errorf("expected 2 emails to be sent, got %d", emailsSent)
	}
	if updatedUser == nil || updatedUser.Email.String() != "newemail@example.com" {
		t.Error("expected user email to be updated in repository")
	}
}

func TestConfirmEmailChange_InvalidCode(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		return activeUser, nil
	}

	emailChangeRepo := &mockEmailChangeRepo{
		findByCodeAndUserIDFn: func(ctx context.Context, code string, userID string) (*domain.EmailChangeRequest, error) {
			return nil, nil // Request not found by code
		},
	}

	mockEmailSender := &mockEmailSender{}

	uc := NewConfirmEmailChangeUseCase(userRepo, emailChangeRepo, mockEmailSender)

	_, err := uc.Execute(context.Background(), ConfirmEmailChangeInput{
		UserID: "user-1",
		Code:   "badcod",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != "INVALID_CODE" {
		t.Errorf("expected INVALID_CODE error, got %v", err)
	}
	if appErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", appErr.StatusCode)
	}
}

func TestConfirmEmailChange_ExpiredCode(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		return activeUser, nil
	}

	mockEmail, _ := domain.NewEmail("newemail@example.com")
	req := &domain.EmailChangeRequest{
		ID:        "req-1",
		UserID:    "user-1",
		NewEmail:  mockEmail,
		Code:      "123456",
		Status:    domain.StatusPending,
		ExpiresAt: time.Now().Add(-10 * time.Minute), // EXPIRED!
	}

	emailChangeRepo := &mockEmailChangeRepo{
		findByCodeAndUserIDFn: func(ctx context.Context, code string, userID string) (*domain.EmailChangeRequest, error) {
			return req, nil
		},
	}
	
	mockEmailSender := &mockEmailSender{}

	uc := NewConfirmEmailChangeUseCase(userRepo, emailChangeRepo, mockEmailSender)

	_, err := uc.Execute(context.Background(), ConfirmEmailChangeInput{
		UserID: "user-1",
		Code:   "123456",
	})

	if err == nil {
		t.Fatal("expected error for expired request, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != "INVALID_CODE" {
		t.Errorf("expected INVALID_CODE error, got %v", err)
	}
}

func TestConfirmEmailChange_DuplicateEmailAtConfirmation(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		return activeUser, nil
	}

	// Wait! While it was pending, someone else registered the new email!
	userRepo.existsByEmailFn = func(_ context.Context, _ domain.Email) (bool, error) {
		return true, nil 
	}

	mockEmail, _ := domain.NewEmail("stolen@example.com")
	req := &domain.EmailChangeRequest{
		ID:        "req-1",
		UserID:    "user-1",
		NewEmail:  mockEmail,
		Code:      "123456",
		Status:    domain.StatusPending,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	emailChangeRepo := &mockEmailChangeRepo{
		findByCodeAndUserIDFn: func(ctx context.Context, code string, userID string) (*domain.EmailChangeRequest, error) {
			return req, nil
		},
	}
	
	mockEmailSender := &mockEmailSender{}

	uc := NewConfirmEmailChangeUseCase(userRepo, emailChangeRepo, mockEmailSender)

	_, err := uc.Execute(context.Background(), ConfirmEmailChangeInput{
		UserID: "user-1",
		Code:   "123456",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != "EMAIL_ALREADY_EXISTS" {
		t.Errorf("expected EMAIL_ALREADY_EXISTS error, got %v", err)
	}
}
