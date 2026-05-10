package user

import (
	"context"
	"errors"
	"testing"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
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
	activeUser := newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive)

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
			if req.UserID() != "user-1" {
				t.Errorf("expected user-1, got %s", req.UserID())
			}
			if len(req.VerificationCode()) != 6 {
				t.Errorf("expected 6-digit code, got %s", req.VerificationCode())
			}
			return nil
		},
	}

	emailSent := false
	mockEmail := &mockEmailSender{
		sendFn: func(ctx context.Context, msg appshared.EmailMessage) error {
			emailSent = true
			if msg.To != "user-1@example.com" {
				t.Errorf("expected user@example.com, got %s", msg.To)
			}
			return nil
		},
	}

	uc := NewRequestDeactivationUseCase(userRepo, deactRepo, mockEmail)
	err := uc.Execute(context.Background(), RequestDeactivationInput{UserID: "user-1"})
	
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if !emailSent {
		t.Error("expected email to be sent")
	}
}

func TestRequestDeactivation_AdminForbidden(t *testing.T) {
	userRepo := newNoConflictRepo()
	adminUser := newUserWithRole("admin-1", domainShared.RoleAdmin, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		return adminUser, nil
	}

	uc := NewRequestDeactivationUseCase(userRepo, &mockDeactivationRepo{}, &mockEmailSender{})
	err := uc.Execute(context.Background(), RequestDeactivationInput{UserID: "admin-1"})
	
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != ErrCodeForbidden {
		t.Errorf("expected FORBIDDEN error, got %v", err)
	}
}

func TestRequestDeactivation_UserNotFound(t *testing.T) {
	userRepo := newNoConflictRepo()
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		return nil, nil
	}

	uc := NewRequestDeactivationUseCase(userRepo, &mockDeactivationRepo{}, &mockEmailSender{})
	err := uc.Execute(context.Background(), RequestDeactivationInput{UserID: "non-existent"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != ErrCodeUserNotFound {
		t.Errorf("expected NOT_FOUND error, got %v", err)
	}
}

func TestRequestDeactivation_UserRepoError(t *testing.T) {
	userRepo := newNoConflictRepo()
	userRepo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return nil, errors.New("db down")
	}

	uc := NewRequestDeactivationUseCase(userRepo, &mockDeactivationRepo{}, &mockEmailSender{})
	err := uc.Execute(context.Background(), RequestDeactivationInput{UserID: "user-1"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected INTERNAL_ERROR, got %v", err)
	}
}

func TestRequestDeactivation_InvalidatePendingError(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) { return activeUser, nil }

	deactRepo := &mockDeactivationRepo{
		invalidatePendingByUserIDFn: func(_ context.Context, _ string, _ time.Time) error {
			return errors.New("db error")
		},
	}

	uc := NewRequestDeactivationUseCase(userRepo, deactRepo, &mockEmailSender{})
	err := uc.Execute(context.Background(), RequestDeactivationInput{UserID: "user-1"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected INTERNAL_ERROR, got %v", err)
	}
}

func TestRequestDeactivation_SaveError(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) { return activeUser, nil }

	deactRepo := &mockDeactivationRepo{
		saveFn: func(_ context.Context, _ *domain.DeactivationRequest) error {
			return errors.New("insert failed")
		},
	}

	uc := NewRequestDeactivationUseCase(userRepo, deactRepo, &mockEmailSender{})
	err := uc.Execute(context.Background(), RequestDeactivationInput{UserID: "user-1"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected INTERNAL_ERROR, got %v", err)
	}
}

func TestRequestDeactivation_EmailDeliveryFailure(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) { return activeUser, nil }

	emailSender := &mockEmailSender{
		sendFn: func(_ context.Context, _ appshared.EmailMessage) error {
			return errors.New("smtp error")
		},
	}

	uc := NewRequestDeactivationUseCase(userRepo, &mockDeactivationRepo{}, emailSender)
	err := uc.Execute(context.Background(), RequestDeactivationInput{UserID: "user-1"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != ErrCodeEmailDeliveryFailed {
		t.Errorf("expected EMAIL_DELIVERY_FAILED, got %v", err)
	}
}
