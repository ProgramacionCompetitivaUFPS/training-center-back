package auth

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTRefreshTokenCodec_WrapUnwrap_Roundtrip(t *testing.T) {
	c := NewJWTRefreshTokenCodec("test-secret")

	wrapped, err := c.Wrap(context.Background(), "raw-secret", "user-id")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	secret, userID, err := c.Unwrap(wrapped)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if secret != "raw-secret" {
		t.Errorf("expected secret %q, got %q", "raw-secret", secret)
	}
	if userID != "user-id" {
		t.Errorf("expected userID %q, got %q", "user-id", userID)
	}
}

func TestJWTRefreshTokenCodec_Unwrap_WrongSecret_Rejected(t *testing.T) {
	c := NewJWTRefreshTokenCodec("test-secret")
	other := NewJWTRefreshTokenCodec("other-secret")

	wrapped, err := c.Wrap(context.Background(), "raw-secret", "user-id")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if _, _, err := other.Unwrap(wrapped); err == nil {
		t.Fatal("expected error unwrapping an envelope signed with a different secret")
	}
}

func TestJWTRefreshTokenCodec_Unwrap_Malformed_Rejected(t *testing.T) {
	c := NewJWTRefreshTokenCodec("test-secret")

	if _, _, err := c.Unwrap("not-a-jwt"); err == nil {
		t.Fatal("expected error unwrapping a malformed string")
	}
}

func TestJWTRefreshTokenCodec_Unwrap_AccessTokenShape_Rejected(t *testing.T) {
	// An access token (JWTService.GenerateToken's shape) has no "secret" claim — Unwrap must
	// not treat it as a valid refresh envelope just because the signature checks out.
	c := NewJWTRefreshTokenCodec("test-secret")

	accessLikeClaims := struct {
		Email string `json:"email"`
		Role  string `json:"role"`
		jwt.RegisteredClaims
	}{
		Email: "test@example.com",
		Role:  "CONTESTANT",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-id",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, accessLikeClaims)
	signed, err := token.SignedString(c.secret)
	if err != nil {
		t.Fatalf("failed to build access-token-shaped fixture: %v", err)
	}

	if _, _, err := c.Unwrap(signed); err == nil {
		t.Fatal("expected error unwrapping an access-token-shaped JWT (missing secret claim)")
	}
}

func TestJWTRefreshTokenCodec_Unwrap_WrongAlgorithm_Rejected(t *testing.T) {
	c := NewJWTRefreshTokenCodec("test-secret")

	claims := refreshTokenClaims{
		Secret:           "raw-secret",
		RegisteredClaims: jwt.RegisteredClaims{Subject: "user-id"},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)
	signed, err := token.SignedString(c.secret)
	if err != nil {
		t.Fatalf("failed to build HS384 fixture: %v", err)
	}

	if _, _, err := c.Unwrap(signed); err == nil {
		t.Fatal("expected error unwrapping a token signed with a non-HS256 algorithm")
	}
}
