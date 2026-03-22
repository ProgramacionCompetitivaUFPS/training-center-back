package user

import (
	"context"
	"testing"
	"time"

	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type mockDeactivationRepo struct {
	saveFn                      func(ctx context.Context, req *domain.DeactivationRequest) error
	findByIDFn                  func(ctx context.Context, id string) (*domain.DeactivationRequest, error)
	updateFn                    func(ctx context.Context, req *domain.DeactivationRequest) error
	findPendingByUserIDFn       func(ctx context.Context, userID string) (*domain.DeactivationRequest, error)
	invalidatePendingByUserIDFn func(ctx context.Context, userID string, now time.Time) error
}

func (m *mockDeactivationRepo) Save(ctx context.Context, req *domain.DeactivationRequest) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, req)
	}
	return nil
}
func (m *mockDeactivationRepo) FindByID(ctx context.Context, id string) (*domain.DeactivationRequest, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockDeactivationRepo) Update(ctx context.Context, req *domain.DeactivationRequest) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, req)
	}
	return nil
}
func (m *mockDeactivationRepo) FindPendingByUserID(ctx context.Context, userID string) (*domain.DeactivationRequest, error) {
	if m.findPendingByUserIDFn != nil {
		return m.findPendingByUserIDFn(ctx, userID)
	}
	return nil, nil
}
func (m *mockDeactivationRepo) InvalidatePendingByUserID(ctx context.Context, userID string, now time.Time) error {
	if m.invalidatePendingByUserIDFn != nil {
		return m.invalidatePendingByUserIDFn(ctx, userID, now)
	}
	return nil
}

func TestRequestDeactivation_Success(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	emailStr := "user@example.com"
	em, _ := domain.NewEmail(emailStr)
	activeUser.Email = &em

	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == "user-1" {
			return activeUser, nil
		}
		return nil, nil
	}

	deactRepo := &mockDeactivationRepo{
		invalidatePendingByUserIDFn: func(ctx context.Context, userID string, now time.Time) error {
			return nil
		},
		saveFn: func(ctx context.Context, req *domain.DeactivationRequest) error {
			if req.UserID != "user-1" {
				t.Errorf("expected user-1, got %s", req.UserID)
			}
			if len(req.VerificationCode) != 6 {
				t.Errorf("expected 6-digit code, got %s", req.VerificationCode)
			}
			return nil
		},
	}

	emailSent := false
	mockEmail := &mockEmailSender{
		sendFn: func(ctx context.Context, to, subject, body string) error {
			emailSent = true
			if to != "user@example.com" {
				t.Errorf("expected user@example.com, got %s", to)
			}
			return nil
		},
	}

	uc := NewRequestDeactivationUseCase(userRepo, deactRepo, mockEmail)
	err := uc.Execute(context.Background(), "user-1")
	
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if !emailSent {
		t.Error("expected email to be sent")
	}
}

func TestRequestDeactivation_AdminForbidden(t *testing.T) {
	userRepo := newNoConflictRepo()
	adminUser := newUserWithRole("admin-1", domain.RoleAdmin, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		return adminUser, nil
	}

	uc := NewRequestDeactivationUseCase(userRepo, &mockDeactivationRepo{}, &mockEmailSender{})
	err := uc.Execute(context.Background(), "admin-1")
	
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != "FORBIDDEN" {
		t.Errorf("expected FORBIDDEN error, got %v", err)
	}
}

func TestRequestDeactivation_UserNotFound(t *testing.T) {
	userRepo := newNoConflictRepo()
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		return nil, nil
	}

	uc := NewRequestDeactivationUseCase(userRepo, &mockDeactivationRepo{}, &mockEmailSender{})
	err := uc.Execute(context.Background(), "non-existent")
	
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != "NOT_FOUND" {
		t.Errorf("expected NOT_FOUND error, got %v", err)
	}
}
