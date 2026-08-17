package user

import (
	"context"
	"testing"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestUnlinkGoogleIdentity_LinkedAccount_DeletesIdentity(t *testing.T) {
	deleteCalledFor := ""
	oauthRepo := &mockOAuthIdentityRepository{
		findByUserIDFn: func(_ context.Context, _ string, _ domain.OAuthProvider) (*domain.OAuthIdentity, error) {
			identity, err := domain.NewOAuthIdentity("identity-1", "user-abc", domain.OAuthProviderGoogle, "google-sub-1", time.Now())
			if err != nil {
				t.Fatalf("unexpected error building test identity: %v", err)
			}
			return identity, nil
		},
		deleteByUserIDFn: func(_ context.Context, userID string, _ domain.OAuthProvider) error {
			deleteCalledFor = userID
			return nil
		},
	}
	uc := NewUnlinkGoogleIdentityUseCase(&mockUserRepository{}, oauthRepo, &mockEmailSender{})

	err := uc.Execute(context.Background(), UnlinkGoogleIdentityInput{UserID: "user-abc"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if deleteCalledFor != "user-abc" {
		t.Errorf("expected DeleteByUserID called with %q, got %q", "user-abc", deleteCalledFor)
	}
}

func TestUnlinkGoogleIdentity_LinkedAccount_SendsSecurityAlertEmail(t *testing.T) {
	oauthRepo := &mockOAuthIdentityRepository{
		findByUserIDFn: func(_ context.Context, _ string, _ domain.OAuthProvider) (*domain.OAuthIdentity, error) {
			identity, err := domain.NewOAuthIdentity("identity-1", "user-uuid-123", domain.OAuthProviderGoogle, "google-sub-1", time.Now())
			if err != nil {
				t.Fatalf("unexpected error building test identity: %v", err)
			}
			return identity, nil
		},
		deleteByUserIDFn: func(_ context.Context, _ string, _ domain.OAuthProvider) error {
			return nil
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
	deleteCalled := false
	oauthRepo := &mockOAuthIdentityRepository{
		findByUserIDFn: func(_ context.Context, _ string, _ domain.OAuthProvider) (*domain.OAuthIdentity, error) {
			return nil, nil
		},
		deleteByUserIDFn: func(_ context.Context, _ string, _ domain.OAuthProvider) error {
			deleteCalled = true
			return nil
		},
	}
	uc := NewUnlinkGoogleIdentityUseCase(&mockUserRepository{}, oauthRepo, &mockEmailSender{})

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
	if deleteCalled {
		t.Error("expected DeleteByUserID NOT to be called when nothing is linked")
	}
}

func TestUnlinkGoogleIdentity_FindError_PropagatesError(t *testing.T) {
	oauthRepo := &mockOAuthIdentityRepository{
		findByUserIDFn: func(_ context.Context, _ string, _ domain.OAuthProvider) (*domain.OAuthIdentity, error) {
			return nil, apperror.NewInternal()
		},
	}
	uc := NewUnlinkGoogleIdentityUseCase(&mockUserRepository{}, oauthRepo, &mockEmailSender{})

	err := uc.Execute(context.Background(), UnlinkGoogleIdentityInput{UserID: "user-abc"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if _, ok := err.(*apperror.AppError); !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
}
