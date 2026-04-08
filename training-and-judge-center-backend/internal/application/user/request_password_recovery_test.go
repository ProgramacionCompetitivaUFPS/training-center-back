package user

import (
	"context"
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/notification"
	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type mockPasswordRecoveryRepo struct {
	saveFn                      func(ctx context.Context, req *domain.PasswordRecoveryRequest) error
	findByIDFn                  func(ctx context.Context, id string) (*domain.PasswordRecoveryRequest, error)
	updateFn                    func(ctx context.Context, req *domain.PasswordRecoveryRequest) error
	findPendingByUserIDFn       func(ctx context.Context, userID string) (*domain.PasswordRecoveryRequest, error)
	invalidatePendingByUserIDFn func(ctx context.Context, userID string, now time.Time) error
}

func (m *mockPasswordRecoveryRepo) Save(ctx context.Context, req *domain.PasswordRecoveryRequest) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, req)
	}
	return nil
}
func (m *mockPasswordRecoveryRepo) FindByID(ctx context.Context, id string) (*domain.PasswordRecoveryRequest, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockPasswordRecoveryRepo) Update(ctx context.Context, req *domain.PasswordRecoveryRequest) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, req)
	}
	return nil
}
func (m *mockPasswordRecoveryRepo) FindPendingByUserID(ctx context.Context, userID string) (*domain.PasswordRecoveryRequest, error) {
	if m.findPendingByUserIDFn != nil {
		return m.findPendingByUserIDFn(ctx, userID)
	}
	return nil, nil
}
func (m *mockPasswordRecoveryRepo) InvalidatePendingByUserID(ctx context.Context, userID string, now time.Time) error {
	if m.invalidatePendingByUserIDFn != nil {
		return m.invalidatePendingByUserIDFn(ctx, userID, now)
	}
	return nil
}

func TestRequestPasswordRecovery_Success(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	userRepo.findByEmailFn = func(_ context.Context, email domain.Email) (*domain.User, error) {
		if email.String() == "user-1@example.com" {
			return activeUser, nil
		}
		return nil, nil
	}

	recoveryRepo := &mockPasswordRecoveryRepo{
		invalidatePendingByUserIDFn: func(ctx context.Context, userID string, now time.Time) error {
			return nil
		},
		saveFn: func(ctx context.Context, req *domain.PasswordRecoveryRequest) error {
			if req.UserID() != "user-1" {
				t.Errorf("expected user-1, got %s", req.UserID())
			}
			if len(req.Code()) != 6 {
				t.Errorf("expected 6-digit code, got %s", req.Code())
			}
			return nil
		},
	}

	emailSent := false
	mockEmail := &mockEmailSender{
		sendFn: func(ctx context.Context, msg notification.EmailMessage) error {
			emailSent = true
			if msg.To != "user-1@example.com" {
				t.Errorf("expected user@example.com, got %s", msg.To)
			}
			return nil
		},
	}

	uc := NewRequestPasswordRecoveryUseCase(userRepo, recoveryRepo, mockEmail, &mockRateLimiter{})
	err := uc.Execute(context.Background(), RequestPasswordRecoveryInput{Email: "user-1@example.com"})
	
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if !emailSent {
		t.Error("expected email to be sent")
	}
}

func TestRequestPasswordRecovery_AmbiguousResponseWhenNotFound(t *testing.T) {
	userRepo := newNoConflictRepo()
	userRepo.findByEmailFn = func(_ context.Context, email domain.Email) (*domain.User, error) {
		return nil, nil // Not found
	}

	recoveryRepo := &mockPasswordRecoveryRepo{} // shouldn't be called
	emailSent := false
	mockEmail := &mockEmailSender{
		sendFn: func(ctx context.Context, msg notification.EmailMessage) error {
			emailSent = true
			return nil
		},
	}

	uc := NewRequestPasswordRecoveryUseCase(userRepo, recoveryRepo, mockEmail, &mockRateLimiter{})
	err := uc.Execute(context.Background(), RequestPasswordRecoveryInput{Email: "unknown@example.com"})
	
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if emailSent {
		t.Error("expected NO email to be sent")
	}
}

func TestRequestPasswordRecovery_InvalidEmail(t *testing.T) {
	uc := NewRequestPasswordRecoveryUseCase(newNoConflictRepo(), &mockPasswordRecoveryRepo{}, &mockEmailSender{}, &mockRateLimiter{})
	err := uc.Execute(context.Background(), RequestPasswordRecoveryInput{Email: "not-an-email"})
	
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != "VALIDATION_ERROR" {
		t.Errorf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestRequestPasswordRecovery_EmailSendFailNoError(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	userRepo.findByEmailFn = func(_ context.Context, email domain.Email) (*domain.User, error) {
		if email.String() == "user-1@example.com" {
			return activeUser, nil
		}
		return nil, nil
	}

	recoveryRepo := &mockPasswordRecoveryRepo{
		invalidatePendingByUserIDFn: func(ctx context.Context, userID string, now time.Time) error {
			return nil
		},
		saveFn: func(ctx context.Context, req *domain.PasswordRecoveryRequest) error {
			return nil
		},
	}

	mockEmail := &mockEmailSender{
		sendFn: func(ctx context.Context, msg notification.EmailMessage) error {
			return apperror.NewInternal()
		},
	}

	uc := NewRequestPasswordRecoveryUseCase(userRepo, recoveryRepo, mockEmail, &mockRateLimiter{})
	err := uc.Execute(context.Background(), RequestPasswordRecoveryInput{Email: "user-1@example.com"})

	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestRequestPasswordRecovery_EmailSendFail_InvalidatesOrphanedCode(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	userRepo.findByEmailFn = func(_ context.Context, email domain.Email) (*domain.User, error) {
		return activeUser, nil
	}

	invalidateCalls := 0
	recoveryRepo := &mockPasswordRecoveryRepo{
		invalidatePendingByUserIDFn: func(ctx context.Context, userID string, now time.Time) error {
			invalidateCalls++
			return nil
		},
		saveFn: func(ctx context.Context, req *domain.PasswordRecoveryRequest) error {
			return nil
		},
	}

	mockEmail := &mockEmailSender{
		sendFn: func(ctx context.Context, msg notification.EmailMessage) error {
			return apperror.NewInternal()
		},
	}

	uc := NewRequestPasswordRecoveryUseCase(userRepo, recoveryRepo, mockEmail, &mockRateLimiter{})
	err := uc.Execute(context.Background(), RequestPasswordRecoveryInput{Email: "user-1@example.com"})

	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if invalidateCalls != 2 {
		t.Errorf("expected InvalidatePendingByUserID to be called twice (pre-save + email-fail), got %d", invalidateCalls)
	}
}
