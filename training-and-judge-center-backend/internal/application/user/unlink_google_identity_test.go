package user

import (
	"context"
	"testing"
	"time"

	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestUnlinkGoogleIdentity_LinkedAccount_DeletesIdentity(t *testing.T) {
	deleteCalledFor := ""
	oauthRepo := &mockOAuthIdentityRepository{
		findByUserIDFn: func(_ context.Context, _ string) (*domain.OAuthIdentity, error) {
			identity, err := domain.NewOAuthIdentity("identity-1", "user-abc", domain.OAuthProviderGoogle, "google-sub-1", time.Now())
			if err != nil {
				t.Fatalf("unexpected error building test identity: %v", err)
			}
			return identity, nil
		},
		deleteByUserIDFn: func(_ context.Context, userID string) error {
			deleteCalledFor = userID
			return nil
		},
	}
	uc := NewUnlinkGoogleIdentityUseCase(oauthRepo)

	err := uc.Execute(context.Background(), UnlinkGoogleIdentityInput{UserID: "user-abc"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if deleteCalledFor != "user-abc" {
		t.Errorf("expected DeleteByUserID called with %q, got %q", "user-abc", deleteCalledFor)
	}
}

func TestUnlinkGoogleIdentity_NoLinkedAccount_ReturnsNotFound(t *testing.T) {
	deleteCalled := false
	oauthRepo := &mockOAuthIdentityRepository{
		findByUserIDFn: func(_ context.Context, _ string) (*domain.OAuthIdentity, error) {
			return nil, nil
		},
		deleteByUserIDFn: func(_ context.Context, _ string) error {
			deleteCalled = true
			return nil
		},
	}
	uc := NewUnlinkGoogleIdentityUseCase(oauthRepo)

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
		findByUserIDFn: func(_ context.Context, _ string) (*domain.OAuthIdentity, error) {
			return nil, apperror.NewInternal()
		},
	}
	uc := NewUnlinkGoogleIdentityUseCase(oauthRepo)

	err := uc.Execute(context.Background(), UnlinkGoogleIdentityInput{UserID: "user-abc"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if _, ok := err.(*apperror.AppError); !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
}
