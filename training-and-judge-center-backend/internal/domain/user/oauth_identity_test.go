package user_test

import (
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/user"
)

func TestNewOAuthIdentity_Valid(t *testing.T) {
	now := time.Now()
	identity, err := user.NewOAuthIdentity("identity-1", "user-1", user.OAuthProviderGoogle, "google-sub-123", now)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if identity.ID() != "identity-1" {
		t.Errorf("expected ID %q, got %q", "identity-1", identity.ID())
	}
	if identity.UserID() != "user-1" {
		t.Errorf("expected UserID %q, got %q", "user-1", identity.UserID())
	}
	if identity.Provider() != user.OAuthProviderGoogle {
		t.Errorf("expected provider %q, got %q", user.OAuthProviderGoogle, identity.Provider())
	}
	if identity.ProviderUserID() != "google-sub-123" {
		t.Errorf("expected ProviderUserID %q, got %q", "google-sub-123", identity.ProviderUserID())
	}
	if !identity.CreatedAt().Equal(now.UTC()) {
		t.Errorf("expected CreatedAt %v, got %v", now.UTC(), identity.CreatedAt())
	}
}

func TestNewOAuthIdentity_Invalid(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name           string
		id             string
		userID         string
		providerUserID string
	}{
		{"empty id", "", "user-1", "google-sub-123"},
		{"empty userID", "identity-1", "", "google-sub-123"},
		{"empty providerUserID", "identity-1", "user-1", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := user.NewOAuthIdentity(tt.id, tt.userID, user.OAuthProviderGoogle, tt.providerUserID, now)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestRestoreOAuthIdentity(t *testing.T) {
	createdAt := time.Now().UTC()
	identity := user.RestoreOAuthIdentity("identity-1", "user-1", user.OAuthProviderGoogle, "google-sub-123", createdAt)

	if identity.ID() != "identity-1" {
		t.Errorf("expected ID %q, got %q", "identity-1", identity.ID())
	}
	if identity.UserID() != "user-1" {
		t.Errorf("expected UserID %q, got %q", "user-1", identity.UserID())
	}
	if identity.Provider() != user.OAuthProviderGoogle {
		t.Errorf("expected provider %q, got %q", user.OAuthProviderGoogle, identity.Provider())
	}
	if identity.ProviderUserID() != "google-sub-123" {
		t.Errorf("expected ProviderUserID %q, got %q", "google-sub-123", identity.ProviderUserID())
	}
	if !identity.CreatedAt().Equal(createdAt) {
		t.Errorf("expected CreatedAt %v, got %v", createdAt, identity.CreatedAt())
	}
}
