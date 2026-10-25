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

	uc := NewRequestDeactivationUseCase(userRepo, deactRepo, mockEmail, &mockRateLimiter{})
	err := uc.Execute(context.Background(), RequestDeactivationInput{UserID: "user-1", Password: "Secret1!"})
	
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

	uc := NewRequestDeactivationUseCase(userRepo, &mockDeactivationRepo{}, &mockEmailSender{}, &mockRateLimiter{})
	err := uc.Execute(context.Background(), RequestDeactivationInput{UserID: "admin-1"})
	
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != ErrCodeAdminCannotRequestDeactivation {
		t.Errorf("expected FORBIDDEN error, got %v", err)
	}
}

func TestRequestDeactivation_UserNotFound(t *testing.T) {
	userRepo := newNoConflictRepo()
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		return nil, nil
	}

	uc := NewRequestDeactivationUseCase(userRepo, &mockDeactivationRepo{}, &mockEmailSender{}, &mockRateLimiter{})
	err := uc.Execute(context.Background(), RequestDeactivationInput{UserID: "non-existent"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != domain.ErrCodeUserNotFound {
		t.Errorf("expected NOT_FOUND error, got %v", err)
	}
}

func TestRequestDeactivation_UserRepoError(t *testing.T) {
	userRepo := newNoConflictRepo()
	userRepo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return nil, apperror.NewInternal()
	}

	uc := NewRequestDeactivationUseCase(userRepo, &mockDeactivationRepo{}, &mockEmailSender{}, &mockRateLimiter{})
	err := uc.Execute(context.Background(), RequestDeactivationInput{UserID: "user-1", Password: "Secret1!"})

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
			return apperror.NewInternal()
		},
	}

	uc := NewRequestDeactivationUseCase(userRepo, deactRepo, &mockEmailSender{}, &mockRateLimiter{})
	err := uc.Execute(context.Background(), RequestDeactivationInput{UserID: "user-1", Password: "Secret1!"})

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
			return apperror.NewInternal()
		},
	}

	uc := NewRequestDeactivationUseCase(userRepo, deactRepo, &mockEmailSender{}, &mockRateLimiter{})
	err := uc.Execute(context.Background(), RequestDeactivationInput{UserID: "user-1", Password: "Secret1!"})

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

	uc := NewRequestDeactivationUseCase(userRepo, &mockDeactivationRepo{}, emailSender, &mockRateLimiter{})
	err := uc.Execute(context.Background(), RequestDeactivationInput{UserID: "user-1", Password: "Secret1!"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != ErrCodeEmailDeliveryFailed {
		t.Errorf("expected EMAIL_DELIVERY_FAILED, got %v", err)
	}
}

// ── Password check (FL-USR-06) ────────────────────────────────────────────────

func activeUserRepo(u *domain.User) *mockUserRepository {
	repo := newNoConflictRepo()
	repo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == u.ID() {
			return u, nil
		}
		return nil, nil
	}
	return repo
}

func TestRequestDeactivation_WrongPassword_RejectedAndNoEmail(t *testing.T) {
	u := newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive)
	emailSent := false
	uc := NewRequestDeactivationUseCase(activeUserRepo(u), &mockDeactivationRepo{},
		&mockEmailSender{sendFn: func(context.Context, appshared.EmailMessage) error { emailSent = true; return nil }},
		&mockRateLimiter{})

	err := uc.Execute(context.Background(), RequestDeactivationInput{UserID: "user-1", Password: "wrong-password"})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindValidation {
		t.Fatalf("expected a validation error for a wrong password, got %v", err)
	}
	if emailSent {
		t.Error("no code must be emailed when the password is wrong")
	}
}

func TestRequestDeactivation_GoogleOnlyAccount_NoPasswordNeeded(t *testing.T) {
	email := "google-user@example.com"
	u := domain.RestoreUser("user-g", &email, "", "Google User", "google_user", "", "", "",
		domainShared.RoleContestant.String(), domain.StatusActive.String(), time.Now(), nil, nil)
	uc := NewRequestDeactivationUseCase(activeUserRepo(u), &mockDeactivationRepo{}, &mockEmailSender{}, &mockRateLimiter{})

	if err := uc.Execute(context.Background(), RequestDeactivationInput{UserID: "user-g"}); err != nil {
		t.Fatalf("expected an account without password to request deactivation with the code alone, got %v", err)
	}
}

func TestRequestDeactivation_PasswordAttemptsAreRateLimited(t *testing.T) {
	u := newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive)
	var key string
	var max int
	rl := &mockRateLimiter{allowFn: func(_ context.Context, k string, m int, _ time.Duration) (bool, error) {
		key, max = k, m
		return false, nil
	}}
	uc := NewRequestDeactivationUseCase(activeUserRepo(u), &mockDeactivationRepo{}, &mockEmailSender{}, rl)

	err := uc.Execute(context.Background(), RequestDeactivationInput{UserID: "user-1", Password: "Secret1!"})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindTooManyRequests {
		t.Fatalf("expected 429 when the limiter refuses, got %v", err)
	}
	if key != "rate_limit:request_deactivation:user-1" || max != 5 {
		t.Errorf("unexpected limiter key/max: %q / %d", key, max)
	}
}
