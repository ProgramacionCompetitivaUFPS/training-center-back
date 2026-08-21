package user

import (
	"context"
	"testing"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestUnlinkGoogleIdentity_LinkedAccount_DeletesIdentity(t *testing.T) {
	deleteCalledFor := ""
	oauthRepo := &mockOAuthIdentityRepository{
		deleteByUserIDFn: func(_ context.Context, userID string, _ domain.OAuthProvider) (bool, error) {
			deleteCalledFor = userID
			return true, nil
		},
	}
	userRepo := &mockUserRepository{
		findByIDFn: func(_ context.Context, _ string) (*domain.User, error) {
			return newActiveUser(), nil
		},
	}
	uc := NewUnlinkGoogleIdentityUseCase(userRepo, oauthRepo, &mockEmailSender{})

	err := uc.Execute(context.Background(), UnlinkGoogleIdentityInput{UserID: "user-abc"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if deleteCalledFor != "user-abc" {
		t.Errorf("expected DeleteByUserID called with %q, got %q", "user-abc", deleteCalledFor)
	}
}

func TestUnlinkGoogleIdentity_NoPasswordSet_ReturnsConflict(t *testing.T) {
	deleteCalled := false
	oauthRepo := &mockOAuthIdentityRepository{
		deleteByUserIDFn: func(_ context.Context, _ string, _ domain.OAuthProvider) (bool, error) {
			deleteCalled = true
			return true, nil
		},
	}
	userRepo := &mockUserRepository{
		findByIDFn: func(_ context.Context, _ string) (*domain.User, error) {
			return newGoogleOnlyUser("user-abc", domain.StatusActive), nil
		},
	}
	uc := NewUnlinkGoogleIdentityUseCase(userRepo, oauthRepo, &mockEmailSender{})

	err := uc.Execute(context.Background(), UnlinkGoogleIdentityInput{UserID: "user-abc"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != ErrCodeCannotUnlinkLastCredential {
		t.Errorf("expected code %q, got %q", ErrCodeCannotUnlinkLastCredential, appErr.Code)
	}
	if appErr.Kind != apperror.KindConflict {
		t.Errorf("expected kind CONFLICT, got %s", appErr.Kind)
	}
	if deleteCalled {
		t.Error("expected DeleteByUserID NOT to be called when the user has no password")
	}
}

func TestUnlinkGoogleIdentity_LinkedAccount_SendsSecurityAlertEmail(t *testing.T) {
	oauthRepo := &mockOAuthIdentityRepository{
		deleteByUserIDFn: func(_ context.Context, _ string, _ domain.OAuthProvider) (bool, error) {
			return true, nil
		},
	}
	userRepo := &mockUserRepository{
		findByIDFn: func(_ context.Context, _ string) (*domain.User, error) {
			return newActiveUser(), nil
		},
	}
	var sentTo, sentSubject string
	emailSender := &mockEmailSender{
		sendFn: func(_ context.Context, msg appshared.EmailMessage) error {
			sentTo = msg.To
			sentSubject = msg.Subject
			return nil
		},
	}
	uc := NewUnlinkGoogleIdentityUseCase(userRepo, oauthRepo, emailSender)

	err := uc.Execute(context.Background(), UnlinkGoogleIdentityInput{UserID: "user-uuid-123"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sentTo != "test@example.com" {
		t.Errorf("expected notification email to %q, got %q", "test@example.com", sentTo)
	}
	if sentSubject != "Security Alert: Google Account Unlinked" {
		t.Errorf("expected subject %q, got %q", "Security Alert: Google Account Unlinked", sentSubject)
	}
}

func TestUnlinkGoogleIdentity_NoLinkedAccount_ReturnsNotFound(t *testing.T) {
	oauthRepo := &mockOAuthIdentityRepository{
		deleteByUserIDFn: func(_ context.Context, _ string, _ domain.OAuthProvider) (bool, error) {
			return false, nil
		},
	}
	userRepo := &mockUserRepository{
		findByIDFn: func(_ context.Context, _ string) (*domain.User, error) {
			return newActiveUser(), nil
		},
	}
	uc := NewUnlinkGoogleIdentityUseCase(userRepo, oauthRepo, &mockEmailSender{})

	err := uc.Execute(context.Background(), UnlinkGoogleIdentityInput{UserID: "user-abc"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != domain.ErrCodeOAuthIdentityNotFound {
		t.Errorf("expected code %q, got %q", domain.ErrCodeOAuthIdentityNotFound, appErr.Code)
	}
	if appErr.Kind != apperror.KindNotFound {
		t.Errorf("expected kind NOT_FOUND, got %s", appErr.Kind)
	}
}

// A concurrent unlink request can delete the row first: the DELETE this
// request runs affects zero rows, and DeleteByUserID reports that atomically
// instead of racing a separate FindByUserID check against it.
func TestUnlinkGoogleIdentity_ConcurrentUnlink_ReturnsNotFoundWithoutEmail(t *testing.T) {
	oauthRepo := &mockOAuthIdentityRepository{
		deleteByUserIDFn: func(_ context.Context, _ string, _ domain.OAuthProvider) (bool, error) {
			return false, nil
		},
	}
	userRepo := &mockUserRepository{
		findByIDFn: func(_ context.Context, _ string) (*domain.User, error) {
			return newActiveUser(), nil
		},
	}
	emailSent := false
	emailSender := &mockEmailSender{
		sendFn: func(_ context.Context, _ appshared.EmailMessage) error {
			emailSent = true
			return nil
		},
	}
	uc := NewUnlinkGoogleIdentityUseCase(userRepo, oauthRepo, emailSender)

	err := uc.Execute(context.Background(), UnlinkGoogleIdentityInput{UserID: "user-abc"})
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != domain.ErrCodeOAuthIdentityNotFound {
		t.Errorf("expected code %q, got %q", domain.ErrCodeOAuthIdentityNotFound, appErr.Code)
	}
	if emailSent {
		t.Error("expected no security-alert email when this request did not actually delete anything")
	}
}

func TestUnlinkGoogleIdentity_DeleteError_PropagatesError(t *testing.T) {
	oauthRepo := &mockOAuthIdentityRepository{
		deleteByUserIDFn: func(_ context.Context, _ string, _ domain.OAuthProvider) (bool, error) {
			return false, apperror.NewInternal()
		},
	}
	userRepo := &mockUserRepository{
		findByIDFn: func(_ context.Context, _ string) (*domain.User, error) {
			return newActiveUser(), nil
		},
	}
	uc := NewUnlinkGoogleIdentityUseCase(userRepo, oauthRepo, &mockEmailSender{})

	err := uc.Execute(context.Background(), UnlinkGoogleIdentityInput{UserID: "user-abc"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if _, ok := err.(*apperror.AppError); !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
}

func TestUnlinkGoogleIdentity_UserNotFound_ReturnsNotFound(t *testing.T) {
	userRepo := &mockUserRepository{
		findByIDFn: func(_ context.Context, _ string) (*domain.User, error) {
			return nil, nil
		},
	}
	uc := NewUnlinkGoogleIdentityUseCase(userRepo, &mockOAuthIdentityRepository{}, &mockEmailSender{})

	err := uc.Execute(context.Background(), UnlinkGoogleIdentityInput{UserID: "user-abc"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != domain.ErrCodeUserNotFound {
		t.Errorf("expected code %q, got %q", domain.ErrCodeUserNotFound, appErr.Code)
	}
}
