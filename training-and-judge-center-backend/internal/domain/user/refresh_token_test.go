package user_test

import (
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/user"
)

func TestNewRefreshToken(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		userID   string
		familyID string
		hash     string
		wantErr  bool
	}{
		{name: "valid", id: "token-id", userID: "user-id", familyID: "family-id", hash: "hash", wantErr: false},
		{name: "empty id", id: "", userID: "user-id", familyID: "family-id", hash: "hash", wantErr: true},
		{name: "empty user id", id: "token-id", userID: "", familyID: "family-id", hash: "hash", wantErr: true},
		{name: "empty family id", id: "token-id", userID: "user-id", familyID: "", hash: "hash", wantErr: true},
		{name: "empty hash", id: "token-id", userID: "user-id", familyID: "family-id", hash: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := user.NewRefreshToken(tt.id, tt.userID, tt.familyID, tt.hash, nil, nil, false, testNow)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if token.IsRevoked() {
				t.Errorf("expected new token to not be revoked")
			}
			if !token.IssuedAt().Equal(testNow.UTC()) {
				t.Errorf("expected issuedAt %v, got %v", testNow.UTC(), token.IssuedAt())
			}
		})
	}
}

func TestNewRefreshToken_Ceiling(t *testing.T) {
	tests := []struct {
		name            string
		rememberSession bool
		expectedCeiling time.Duration
	}{
		{name: "default ceiling", rememberSession: false, expectedCeiling: user.DefaultSessionCeiling},
		{name: "remember session ceiling", rememberSession: true, expectedCeiling: user.RememberSessionCeiling},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := user.NewRefreshToken("id", "user-id", "family-id", "hash", nil, nil, tt.rememberSession, testNow)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			want := testNow.UTC().Add(tt.expectedCeiling)
			if !token.AbsoluteExpiresAt().Equal(want) {
				t.Errorf("expected absoluteExpiresAt %v, got %v", want, token.AbsoluteExpiresAt())
			}
		})
	}
}

func TestRefreshToken_IsExpired(t *testing.T) {
	tests := []struct {
		name            string
		rememberSession bool
		now             time.Time
		expected        bool
	}{
		{name: "before ceiling", rememberSession: false, now: testNow, expected: false},
		{name: "after ceiling", rememberSession: false, now: testNow.Add(user.DefaultSessionCeiling + time.Second), expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := user.NewRefreshToken("id", "user-id", "family-id", "hash", nil, nil, tt.rememberSession, testNow)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := token.IsExpired(tt.now); got != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRefreshToken_WithinGraceWindow(t *testing.T) {
	tests := []struct {
		name     string
		revoked  bool
		now      time.Time
		expected bool
	}{
		{name: "not revoked", revoked: false, expected: false},
		{name: "revoked within window", revoked: true, now: testNow.Add(5 * time.Second), expected: true},
		{name: "revoked exactly at window boundary", revoked: true, now: testNow.Add(user.RefreshGraceWindow), expected: true},
		{name: "revoked outside window", revoked: true, now: testNow.Add(11 * time.Second), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := user.NewRefreshToken("id", "user-id", "family-id", "hash", nil, nil, false, testNow)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.revoked {
				token.Revoke(testNow, nil)
			}
			if got := token.WithinGraceWindow(tt.now); got != tt.expected {
				t.Errorf("WithinGraceWindow() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRefreshToken_Revoke(t *testing.T) {
	token, err := user.NewRefreshToken("id", "user-id", "family-id", "hash", nil, nil, false, testNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	replacedByID := "new-token-id"
	token.Revoke(testNow, &replacedByID)

	if !token.IsRevoked() {
		t.Errorf("expected token to be revoked")
	}
	if token.RevokedAt() == nil || !token.RevokedAt().Equal(testNow.UTC()) {
		t.Errorf("expected revokedAt %v, got %v", testNow.UTC(), token.RevokedAt())
	}
	if token.ReplacedByID() == nil || *token.ReplacedByID() != replacedByID {
		t.Errorf("expected replacedByID %q, got %v", replacedByID, token.ReplacedByID())
	}
}

func TestRefreshToken_Rotate(t *testing.T) {
	token, err := user.NewRefreshToken("id", "user-id", "family-id", "hash", nil, nil, true, testNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rotateAt := testNow.Add(time.Minute)
	successor, err := token.Rotate("new-id", "new-hash", nil, nil, rotateAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if successor.FamilyID() != token.FamilyID() {
		t.Errorf("expected successor to inherit familyID %q, got %q", token.FamilyID(), successor.FamilyID())
	}
	if !successor.AbsoluteExpiresAt().Equal(token.AbsoluteExpiresAt()) {
		t.Errorf("expected successor to inherit absoluteExpiresAt %v, got %v", token.AbsoluteExpiresAt(), successor.AbsoluteExpiresAt())
	}
	if successor.UserID() != token.UserID() {
		t.Errorf("expected successor to inherit userID %q, got %q", token.UserID(), successor.UserID())
	}
	if token.IsRevoked() {
		t.Errorf("Rotate must not mutate the receiver — persistence decides whether it ends up revoked")
	}
}

func TestRefreshToken_Rotate_AlreadyRevoked(t *testing.T) {
	token, err := user.NewRefreshToken("id", "user-id", "family-id", "hash", nil, nil, false, testNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	token.Revoke(testNow, nil)

	if _, err := token.Rotate("new-id", "new-hash", nil, nil, testNow); err == nil {
		t.Errorf("expected error rotating an already-revoked token")
	}
}

func TestRestoreRefreshToken(t *testing.T) {
	revokedAt := testNow
	replacedByID := "new-token-id"
	userAgent := "Mozilla/5.0"
	ipAddress := "203.0.113.5"

	token := user.RestoreRefreshToken(
		"id", "user-id", "family-id", "hash",
		testNow, testNow.Add(time.Hour),
		&revokedAt, &replacedByID,
		&userAgent, &ipAddress,
	)

	if token.ID() != "id" || token.UserID() != "user-id" || token.FamilyID() != "family-id" {
		t.Errorf("unexpected restored identity fields")
	}
	if !token.IsRevoked() {
		t.Errorf("expected restored token to be revoked")
	}
	if token.UserAgent() == nil || *token.UserAgent() != userAgent {
		t.Errorf("unexpected restored userAgent")
	}
	if token.IPAddress() == nil || *token.IPAddress() != ipAddress {
		t.Errorf("unexpected restored ipAddress")
	}
}

func TestRestoreRefreshToken_NilMetadata(t *testing.T) {
	token := user.RestoreRefreshToken(
		"id", "user-id", "family-id", "hash",
		testNow, testNow.Add(time.Hour),
		nil, nil,
		nil, nil,
	)

	if token.UserAgent() != nil || token.IPAddress() != nil {
		t.Errorf("expected nil metadata to remain nil")
	}
}
