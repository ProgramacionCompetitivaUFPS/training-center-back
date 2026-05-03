package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/application/shared"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestConfirmEmailChange_Success(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == "user-1" {
			return activeUser, nil
		}
		return nil, nil
	}
	var updatedUser *domain.User
	userRepo.updateFn = func(_ context.Context, u *domain.User) error {
		updatedUser = u
		return nil
	}

	mockEmail, _ := domain.NewEmail("newemail@example.com")
	req := domain.RestoreEmailChangeRequest("req-1", "user-1", mockEmail, "123456", domain.StatusPending, time.Now().Add(10 * time.Minute), time.Time{}, nil)

	emailChangeRepo := &mockEmailChangeRepo{
		findByCodeAndUserIDFn: func(ctx context.Context, code string, userID string) (*domain.EmailChangeRequest, error) {
			if code == "123456" && userID == "user-1" {
				return req, nil
			}
			return nil, nil
		},
		updateFn: func(ctx context.Context, r *domain.EmailChangeRequest) error {
			if r.Status() != domain.StatusUsed {
				t.Errorf("expected request status to be set to USED, got %s", r.Status())
			}
			return nil
		},
	}

	emailsSent := 0
	mockEmailSender := &mockEmailSender{
		sendFn: func(ctx context.Context, msg shared.EmailMessage) error {
			emailsSent++
			return nil
		},
	}

	uc := NewConfirmEmailChangeUseCase(userRepo, emailChangeRepo, mockEmailSender, &mockTransactionManager{})

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
	if updatedUser == nil || updatedUser.Email().String() != "newemail@example.com" {
		t.Error("expected user email to be updated in repository")
	}
}

func TestConfirmEmailChange_InvalidCode(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		return activeUser, nil
	}

	emailChangeRepo := &mockEmailChangeRepo{
		findByCodeAndUserIDFn: func(ctx context.Context, code string, userID string) (*domain.EmailChangeRequest, error) {
			return nil, nil // Request not found by code
		},
	}

	mockEmailSender := &mockEmailSender{}

	uc := NewConfirmEmailChangeUseCase(userRepo, emailChangeRepo, mockEmailSender, &mockTransactionManager{})

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
	if appErr.Kind != apperror.KindBadRequest {
		t.Errorf("expected kind BAD_REQUEST, got %s", appErr.Kind)
	}
}

func TestConfirmEmailChange_ExpiredCode(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		return activeUser, nil
	}

	mockEmail, _ := domain.NewEmail("newemail@example.com")
	req := domain.RestoreEmailChangeRequest("req-1", "user-1", mockEmail, "123456", domain.StatusPending, time.Now().Add(-10 * time.Minute), time.Time{}, nil) // EXPIRED!

	emailChangeRepo := &mockEmailChangeRepo{
		findByCodeAndUserIDFn: func(ctx context.Context, code string, userID string) (*domain.EmailChangeRequest, error) {
			return req, nil
		},
	}
	
	mockEmailSender := &mockEmailSender{}

	uc := NewConfirmEmailChangeUseCase(userRepo, emailChangeRepo, mockEmailSender, &mockTransactionManager{})

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
	activeUser := newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		return activeUser, nil
	}
	// While the request was pending, someone else registered the new email.
	// The DB constraint fires on Update and returns ErrEmailConflict.
	userRepo.updateFn = func(_ context.Context, _ *domain.User) error {
		return domain.ErrEmailConflict
	}

	mockEmail, _ := domain.NewEmail("stolen@example.com")
	req := domain.RestoreEmailChangeRequest("req-1", "user-1", mockEmail, "123456", domain.StatusPending, time.Now().Add(10*time.Minute), time.Time{}, nil)

	emailChangeRepo := &mockEmailChangeRepo{
		findByCodeAndUserIDFn: func(ctx context.Context, code string, userID string) (*domain.EmailChangeRequest, error) {
			return req, nil
		},
	}

	uc := NewConfirmEmailChangeUseCase(userRepo, emailChangeRepo, &mockEmailSender{}, &mockTransactionManager{})

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

func TestConfirmEmailChange_UserNotFound(t *testing.T) {
	userRepo := newNoConflictRepo()
	userRepo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return nil, nil
	}

	uc := NewConfirmEmailChangeUseCase(userRepo, &mockEmailChangeRepo{}, &mockEmailSender{}, &mockTransactionManager{})
	_, err := uc.Execute(context.Background(), ConfirmEmailChangeInput{UserID: "user-1", Code: "123456"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != "INVALID_CREDENTIALS" {
		t.Errorf("expected INVALID_CREDENTIALS, got %v", err)
	}
}

func TestConfirmEmailChange_UserRepoError(t *testing.T) {
	userRepo := newNoConflictRepo()
	userRepo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return nil, errors.New("db error")
	}

	uc := NewConfirmEmailChangeUseCase(userRepo, &mockEmailChangeRepo{}, &mockEmailSender{}, &mockTransactionManager{})
	_, err := uc.Execute(context.Background(), ConfirmEmailChangeInput{UserID: "user-1", Code: "123456"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != "INTERNAL_ERROR" {
		t.Errorf("expected INTERNAL_ERROR, got %v", err)
	}
}

func TestConfirmEmailChange_EmailChangeRepoError(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) { return activeUser, nil }

	emailChangeRepo := &mockEmailChangeRepo{
		findByCodeAndUserIDFn: func(_ context.Context, _ string, _ string) (*domain.EmailChangeRequest, error) {
			return nil, errors.New("db error")
		},
	}

	uc := NewConfirmEmailChangeUseCase(userRepo, emailChangeRepo, &mockEmailSender{}, &mockTransactionManager{})
	_, err := uc.Execute(context.Background(), ConfirmEmailChangeInput{UserID: "user-1", Code: "123456"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != "INTERNAL_ERROR" {
		t.Errorf("expected INTERNAL_ERROR, got %v", err)
	}
}

func TestConfirmEmailChange_TxFailure(t *testing.T) {
	userRepo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domainShared.RoleContestant, domain.StatusActive)
	userRepo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) { return activeUser, nil }

	mockEmail, _ := domain.NewEmail("new@example.com")
	req := domain.RestoreEmailChangeRequest("req-1", "user-1", mockEmail, "123456", domain.StatusPending, time.Now().Add(10*time.Minute), time.Time{}, nil)
	emailChangeRepo := &mockEmailChangeRepo{
		findByCodeAndUserIDFn: func(_ context.Context, _ string, _ string) (*domain.EmailChangeRequest, error) {
			return req, nil
		},
	}

	tx := &mockTransactionManager{
		withTxFn: func(_ context.Context, _ func(context.Context) error) error {
			return errors.New("tx error")
		},
	}

	uc := NewConfirmEmailChangeUseCase(userRepo, emailChangeRepo, &mockEmailSender{}, tx)
	_, err := uc.Execute(context.Background(), ConfirmEmailChangeInput{UserID: "user-1", Code: "123456"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != "INTERNAL_ERROR" {
		t.Errorf("expected INTERNAL_ERROR, got %v", err)
	}
}
