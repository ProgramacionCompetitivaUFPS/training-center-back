package user_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/user"
)

func TestNewOAuthProvider(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"google", "GOOGLE", false},
		{"empty", "", true},
		{"unknown", "FACEBOOK", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := user.NewOAuthProvider(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("input %q: expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("input %q: expected no error, got %v", tt.input, err)
			}
			if provider.String() != tt.input {
				t.Errorf("expected %q, got %q", tt.input, provider.String())
			}
		})
	}
}

func TestOAuthProviderGoogle(t *testing.T) {
	if user.OAuthProviderGoogle.String() != "GOOGLE" {
		t.Errorf("expected %q, got %q", "GOOGLE", user.OAuthProviderGoogle.String())
	}
}

func TestRestoreOAuthProvider(t *testing.T) {
	provider := user.RestoreOAuthProvider("GOOGLE")
	if provider != user.OAuthProviderGoogle {
		t.Errorf("expected %v, got %v", user.OAuthProviderGoogle, provider)
	}
}
