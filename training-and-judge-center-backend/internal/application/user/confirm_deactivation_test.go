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

type mockAuditRepo struct {
	saveFn func(ctx context.Context, log *domain.DeactivationAuditLog) error
}

func (m *mockAuditRepo) Save(ctx context.Context, log *domain.DeactivationAuditLog) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, log)
	}
	return nil
}

func TestConfirmDeactivation_Success(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive)

	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == "user-1" {
			return activeUser, nil
		}
		return nil, nil
	}
	userRepo.updateFn = func(ctx context.Context, u *domain.User) error {
		if u.Status() != domain.StatusDeactivated {
			t.Errorf("expected DEACTIVATED status, got %s", u.Status())
		}
		if u.Email().String() != "" {
			t.Errorf("expected Email to be empty after deactivation, got %v", u.Email())
		}
		return nil
	}

	deactRepo := &mockDeactivationRepo{
		findPendingByUserIDFn: func(ctx context.Context, userID string) (*domain.DeactivationRequest, error) {
			return domain.RestoreDeactivationRequest("req-1", "user-1", "123456", time.Now().Add(10*time.Minute), 0, nil, domain.DeactivationStatusPending, time.Time{}, time.Time{}), nil
		},
		updateFn: func(ctx context.Context, req *domain.DeactivationRequest) error {
			if req.Status() != domain.DeactivationStatusConfirmed {
				t.Errorf("expected expected CONFIRMED, got %s", req.Status())
			}
			return nil
		},
	}

	auditCalled := false
	auditRepo := &mockAuditRepo{
		saveFn: func(ctx context.Context, log *domain.DeactivationAuditLog) error {
			auditCalled = true
			if log.OriginalEmail() != "user-1@example.com" {
				t.Errorf("expected audit email to be user@example.com, got %s", log.OriginalEmail())
			}
			return nil
		},
	}

	invCalled := false
	invalidator := &mockSessionInvalidator{
		invalidateAllUserSessionsFn: func(ctx context.Context, userID string, timestamp time.Time) error {
			invCalled = true
			return nil
		},
	}
	
	emailSent := false
	mockEmail := &mockEmailSender{
		sendFn: func(ctx context.Context, msg appshared.EmailMessage) error {
			emailSent = true
			return nil
		},
	}

	uc := NewConfirmDeactivationUseCase(userRepo, deactRepo, auditRepo, mockEmail, invalidator, &mockTransactionManager{})

	_, err := uc.Execute(context.Background(), ConfirmDeactivationInput{
		UserID:    "user-1",
		Code:      "123456",
	})

	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if !invCalled {
		t.Error("expected session invalidator to be called")
	}
	if !auditCalled {
		t.Error("expected audit log to be saved")
	}
	if !emailSent {
		t.Error("expected confirmation email to be sent")
	}
}

func TestConfirmDeactivation_InvalidCode(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		return activeUser, nil
	}

	deactRepo := &mockDeactivationRepo{
		findPendingByUserIDFn: func(ctx context.Context, userID string) (*domain.DeactivationRequest, error) {
			return domain.RestoreDeactivationRequest("req-1", "user-1", "123456", time.Now().Add(10*time.Minute), 0, nil, domain.DeactivationStatusPending, time.Time{}, time.Time{}), nil
		},
		updateFn: func(ctx context.Context, req *domain.DeactivationRequest) error {
			if req.Attempts() != 1 {
				t.Errorf("expected attempts to be incremented to 1, got %d", req.Attempts())
			}
			return nil
		},
	}

	uc := NewConfirmDeactivationUseCase(userRepo, deactRepo, &mockAuditRepo{}, &mockEmailSender{}, &mockSessionInvalidator{}, &mockTransactionManager{})

	_, err := uc.Execute(context.Background(), ConfirmDeactivationInput{
		UserID: "user-1",
		Code:   "000000",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != "INVALID_CODE" {
		t.Errorf("expected INVALID_CODE, got %v", err)
	}
}

func TestConfirmDeactivation_ExpiredCode_UpdateFails_ReturnsInternal(t *testing.T) {
	userRepo := newNoConflictRepo()
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		return newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive), nil
	}

	dbErr := errors.New("connection refused")
	deactRepo := &mockDeactivationRepo{
		findPendingByUserIDFn: func(ctx context.Context, userID string) (*domain.DeactivationRequest, error) {
			return domain.RestoreDeactivationRequest("req-1", "user-1", "123456", time.Now().Add(-1*time.Minute), 0, nil, domain.DeactivationStatusPending, time.Time{}, time.Time{}), nil
		},
		updateFn: func(ctx context.Context, req *domain.DeactivationRequest) error {
			return dbErr
		},
	}

	uc := NewConfirmDeactivationUseCase(userRepo, deactRepo, &mockAuditRepo{}, &mockEmailSender{}, &mockSessionInvalidator{}, &mockTransactionManager{})

	// Act
	_, err := uc.Execute(context.Background(), ConfirmDeactivationInput{UserID: "user-1", Code: "123456"})

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Kind != apperror.KindInternal {
		t.Errorf("expected kind INTERNAL, got %v", err)
	}
}

func TestConfirmDeactivation_InvalidCode_UpdateFails_ReturnsInternal(t *testing.T) {
	userRepo := newNoConflictRepo()
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		return newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive), nil
	}

	dbErr := errors.New("connection refused")
	deactRepo := &mockDeactivationRepo{
		findPendingByUserIDFn: func(ctx context.Context, userID string) (*domain.DeactivationRequest, error) {
			return domain.RestoreDeactivationRequest("req-1", "user-1", "123456", time.Now().Add(10*time.Minute), 0, nil, domain.DeactivationStatusPending, time.Time{}, time.Time{}), nil
		},
		updateFn: func(ctx context.Context, req *domain.DeactivationRequest) error {
			return dbErr
		},
	}

	uc := NewConfirmDeactivationUseCase(userRepo, deactRepo, &mockAuditRepo{}, &mockEmailSender{}, &mockSessionInvalidator{}, &mockTransactionManager{})

	// Act
	_, err := uc.Execute(context.Background(), ConfirmDeactivationInput{UserID: "user-1", Code: "000000"})

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Kind != apperror.KindInternal {
		t.Errorf("expected kind INTERNAL, got %v", err)
	}
}

func TestConfirmDeactivation_BlockedState_UpdateFails_ReturnsInternal(t *testing.T) {
	userRepo := newNoConflictRepo()
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		return newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive), nil
	}

	dbErr := errors.New("connection refused")
	deactRepo := &mockDeactivationRepo{
		findPendingByUserIDFn: func(ctx context.Context, userID string) (*domain.DeactivationRequest, error) {
			return domain.RestoreDeactivationRequest("req-1", "user-1", "123456", time.Now().Add(10*time.Minute), 4, nil, domain.DeactivationStatusPending, time.Time{}, time.Time{}), nil
		},
		updateFn: func(ctx context.Context, req *domain.DeactivationRequest) error {
			return dbErr
		},
	}

	uc := NewConfirmDeactivationUseCase(userRepo, deactRepo, &mockAuditRepo{}, &mockEmailSender{}, &mockSessionInvalidator{}, &mockTransactionManager{})

	// Act
	_, err := uc.Execute(context.Background(), ConfirmDeactivationInput{UserID: "user-1", Code: "000000"})

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Kind != apperror.KindInternal {
		t.Errorf("expected kind INTERNAL, got %v", err)
	}
}

func TestConfirmDeactivation_AuditLogSaveFails_ReturnsNil(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		return activeUser, nil
	}
	userRepo.updateFn = func(ctx context.Context, u *domain.User) error { return nil }

	deactRepo := &mockDeactivationRepo{
		findPendingByUserIDFn: func(ctx context.Context, userID string) (*domain.DeactivationRequest, error) {
			return domain.RestoreDeactivationRequest("req-1", "user-1", "123456", time.Now().Add(10*time.Minute), 0, nil, domain.DeactivationStatusPending, time.Time{}, time.Time{}), nil
		},
		updateFn: func(ctx context.Context, req *domain.DeactivationRequest) error { return nil },
	}

	auditRepo := &mockAuditRepo{
		saveFn: func(ctx context.Context, log *domain.DeactivationAuditLog) error {
			return errors.New("db unavailable")
		},
	}

	uc := NewConfirmDeactivationUseCase(userRepo, deactRepo, auditRepo, &mockEmailSender{}, &mockSessionInvalidator{}, &mockTransactionManager{})

	_, err := uc.Execute(context.Background(), ConfirmDeactivationInput{UserID: "user-1", Code: "123456"})

	if err != nil {
		t.Fatalf("expected nil (fail-safe), got %v", err)
	}
}

func TestConfirmDeactivation_EmailSendFails_ReturnsNil(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		return activeUser, nil
	}
	userRepo.updateFn = func(ctx context.Context, u *domain.User) error { return nil }

	deactRepo := &mockDeactivationRepo{
		findPendingByUserIDFn: func(ctx context.Context, userID string) (*domain.DeactivationRequest, error) {
			return domain.RestoreDeactivationRequest("req-1", "user-1", "123456", time.Now().Add(10*time.Minute), 0, nil, domain.DeactivationStatusPending, time.Time{}, time.Time{}), nil
		},
		updateFn: func(ctx context.Context, req *domain.DeactivationRequest) error { return nil },
	}

	emailSender := &mockEmailSender{
		sendFn: func(ctx context.Context, msg appshared.EmailMessage) error {
			return errors.New("smtp timeout")
		},
	}

	uc := NewConfirmDeactivationUseCase(userRepo, deactRepo, &mockAuditRepo{}, emailSender, &mockSessionInvalidator{}, &mockTransactionManager{})

	_, err := uc.Execute(context.Background(), ConfirmDeactivationInput{UserID: "user-1", Code: "123456"})

	if err != nil {
		t.Fatalf("expected nil (fail-safe), got %v", err)
	}
}

func TestConfirmDeactivation_BlockExpired_MarksExpiredAndReturnsError(t *testing.T) {
	userRepo := newNoConflictRepo()
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		return newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive), nil
	}

	expiredBlockedUntil := time.Now().Add(-1 * time.Minute)
	deactRepo := &mockDeactivationRepo{
		findPendingByUserIDFn: func(ctx context.Context, userID string) (*domain.DeactivationRequest, error) {
			return domain.RestoreDeactivationRequest("req-1", "user-1", "123456", time.Now().Add(10*time.Minute), 5, &expiredBlockedUntil, domain.DeactivationStatusBlocked, time.Time{}, time.Time{}), nil
		},
		updateFn: func(ctx context.Context, req *domain.DeactivationRequest) error {
			if req.Status() != domain.DeactivationStatusExpired {
				t.Errorf("expected EXPIRED status after block expiry, got %s", req.Status())
			}
			return nil
		},
	}

	uc := NewConfirmDeactivationUseCase(userRepo, deactRepo, &mockAuditRepo{}, &mockEmailSender{}, &mockSessionInvalidator{}, &mockTransactionManager{})

	_, err := uc.Execute(context.Background(), ConfirmDeactivationInput{UserID: "user-1", Code: "123456"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != "EXPIRED_CODE" {
		t.Errorf("expected EXPIRED_CODE after block expiry, got %v", err)
	}
}

func TestConfirmDeactivation_BlockExpired_UpdateFails_ReturnsInternal(t *testing.T) {
	userRepo := newNoConflictRepo()
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		return newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive), nil
	}

	expiredBlockedUntil := time.Now().Add(-1 * time.Minute)
	deactRepo := &mockDeactivationRepo{
		findPendingByUserIDFn: func(ctx context.Context, userID string) (*domain.DeactivationRequest, error) {
			return domain.RestoreDeactivationRequest("req-1", "user-1", "123456", time.Now().Add(10*time.Minute), 5, &expiredBlockedUntil, domain.DeactivationStatusBlocked, time.Time{}, time.Time{}), nil
		},
		updateFn: func(ctx context.Context, req *domain.DeactivationRequest) error {
			return errors.New("db unavailable")
		},
	}

	uc := NewConfirmDeactivationUseCase(userRepo, deactRepo, &mockAuditRepo{}, &mockEmailSender{}, &mockSessionInvalidator{}, &mockTransactionManager{})

	_, err := uc.Execute(context.Background(), ConfirmDeactivationInput{UserID: "user-1", Code: "123456"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Kind != apperror.KindInternal {
		t.Errorf("expected kind INTERNAL, got %v", err)
	}
}

func TestConfirmDeactivation_ActiveBlock(t *testing.T) {
	userRepo := newNoConflictRepo()
	userRepo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive), nil
	}

	blockedUntil := time.Now().Add(30 * time.Minute)
	deactRepo := &mockDeactivationRepo{
		findPendingByUserIDFn: func(_ context.Context, _ string) (*domain.DeactivationRequest, error) {
			return domain.RestoreDeactivationRequest("req-1", "user-1", "123456", time.Now().Add(time.Hour), 5, &blockedUntil, domain.DeactivationStatusBlocked, time.Time{}, time.Time{}), nil
		},
	}

	uc := NewConfirmDeactivationUseCase(userRepo, deactRepo, &mockAuditRepo{}, &mockEmailSender{}, &mockSessionInvalidator{}, &mockTransactionManager{})
	_, err := uc.Execute(context.Background(), ConfirmDeactivationInput{UserID: "user-1", Code: "123456"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != "MAX_ATTEMPTS_EXCEEDED" {
		t.Errorf("expected MAX_ATTEMPTS_EXCEEDED, got %v", err)
	}
}

func TestConfirmDeactivation_ExceedAttemptsAndBlock(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		return activeUser, nil
	}

	deactRepo := &mockDeactivationRepo{
		findPendingByUserIDFn: func(ctx context.Context, userID string) (*domain.DeactivationRequest, error) {
			return domain.RestoreDeactivationRequest("req-1", "user-1", "123456", time.Now().Add(10*time.Minute), 4, nil, domain.DeactivationStatusPending, time.Time{}, time.Time{}), nil
		},
		updateFn: func(ctx context.Context, req *domain.DeactivationRequest) error {
			if req.Status() != domain.DeactivationStatusBlocked {
				t.Errorf("expected status to change to BLOCKED, got %s", req.Status())
			}
			if req.BlockedUntil() == nil {
				t.Error("expected blockedUntil to be set")
			}
			return nil
		},
	}

	uc := NewConfirmDeactivationUseCase(userRepo, deactRepo, &mockAuditRepo{}, &mockEmailSender{}, &mockSessionInvalidator{}, &mockTransactionManager{})

	_, err := uc.Execute(context.Background(), ConfirmDeactivationInput{
		UserID: "user-1",
		Code:   "000000",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != "MAX_ATTEMPTS_EXCEEDED" {
		t.Errorf("expected MAX_ATTEMPTS_EXCEEDED, got %v", err)
	}
}
