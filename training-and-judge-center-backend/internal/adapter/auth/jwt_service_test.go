package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/training-judge-center/backend/internal/domain/user"
)

func testUser() *user.User {
	p, _ := user.NewPassword("Secret1!")
	emailStr := "test@example.com"
	return user.RestoreUser(
		"user-id-123",
		&emailStr,
		p.Hash(),
		"Test User",
		"testuser",
		"US", "NY", "MIT",
		"CONTESTANT",
		"ACTIVE",
		time.Now(),
		nil, nil,
	)
}

func TestGenerateToken_ClaimsPresent(t *testing.T) {
	// Arrange
	svc := NewJWTService("test-secret", 1)
	u := testUser()

	// Act
	tokenStr, err := svc.GenerateToken(context.Background(), u)
	if err != nil {
		t.Fatalf("GenerateToken returned unexpected error: %v", err)
	}

	// Assert — parse manually to inspect raw claim names
	token, err := jwt.ParseWithClaims(tokenStr, &jwtCustomClaims{}, func(token *jwt.Token) (any, error) {
		return []byte("test-secret"), nil
	})
	if err != nil {
		t.Fatalf("could not parse generated token: %v", err)
	}
	claims := token.Claims.(*jwtCustomClaims)
	if claims.Subject != u.ID() {
		t.Errorf("sub: got %q, want %q", claims.Subject, u.ID())
	}
	if claims.Email != u.Email().String() {
		t.Errorf("email: got %q, want %q", claims.Email, u.Email().String())
	}
	if claims.Role != u.Role().String() {
		t.Errorf("role: got %q, want %q", claims.Role, u.Role().String())
	}
	if claims.IssuedAt == nil {
		t.Error("iat claim is missing")
	}
	if claims.ExpiresAt == nil {
		t.Error("exp claim is missing")
	}
}

func TestValidateToken_ValidToken(t *testing.T) {
	// Arrange
	svc := NewJWTService("test-secret", 1)
	u := testUser()
	tokenStr, err := svc.GenerateToken(context.Background(), u)
	if err != nil {
		t.Fatalf("GenerateToken returned unexpected error: %v", err)
	}

	// Act
	claims, err := svc.ValidateToken(tokenStr)

	// Assert
	if err != nil {
		t.Fatalf("ValidateToken returned unexpected error: %v", err)
	}
	if claims.UserID != u.ID() {
		t.Errorf("UserID: got %q, want %q", claims.UserID, u.ID())
	}
	if claims.Email != u.Email() {
		t.Errorf("Email: got %v, want %v", claims.Email, u.Email())
	}
	if claims.Role != u.Role() {
		t.Errorf("Role: got %q, want %q", claims.Role, u.Role())
	}
	if claims.IssuedAt.IsZero() {
		t.Error("IssuedAt is zero")
	}
}

func TestGenerateToken_ExpirationHonored(t *testing.T) {
	// Arrange
	svc := NewJWTService("test-secret", 2)
	u := testUser()

	// Act
	tokenStr, err := svc.GenerateToken(context.Background(), u)
	if err != nil {
		t.Fatalf("GenerateToken returned unexpected error: %v", err)
	}

	// Assert — verify exp - iat equals the configured 2-hour duration
	token, err := jwt.ParseWithClaims(tokenStr, &jwtCustomClaims{}, func(token *jwt.Token) (any, error) {
		return []byte("test-secret"), nil
	})
	if err != nil {
		t.Fatalf("could not parse generated token: %v", err)
	}
	claims := token.Claims.(*jwtCustomClaims)
	diff := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time)
	if diff < 2*time.Hour-time.Second || diff > 2*time.Hour+time.Second {
		t.Errorf("expiration duration: got %v, want ~2h", diff)
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	// Arrange — negative expiration produces a token already expired
	svc := NewJWTService("test-secret", -1)
	u := testUser()
	tokenStr, err := svc.GenerateToken(context.Background(), u)
	if err != nil {
		t.Fatalf("GenerateToken returned unexpected error: %v", err)
	}

	// Act
	_, err = svc.ValidateToken(tokenStr)

	// Assert
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

func TestValidateToken_WrongSigningKey(t *testing.T) {
	// Arrange
	generator := NewJWTService("original-secret", 1)
	validator := NewJWTService("different-secret", 1)
	u := testUser()
	tokenStr, _ := generator.GenerateToken(context.Background(), u)

	// Act
	_, err := validator.ValidateToken(tokenStr)

	// Assert
	if err == nil {
		t.Error("expected error for token signed with a different key, got nil")
	}
}

func TestValidateToken_AlgorithmNone(t *testing.T) {
	// Arrange — craft a token with alg:none (algorithm confusion attack)
	svc := NewJWTService("test-secret", 1)
	noneToken := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub":   "attacker",
		"email": "attacker@example.com",
		"role":  "ADMIN",
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, err := noneToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("could not sign none token: %v", err)
	}

	// Act
	_, err = svc.ValidateToken(tokenStr)

	// Assert
	if err == nil {
		t.Error("expected error for alg:none token (algorithm confusion attack), got nil")
	}
}

func TestValidateToken_MalformedToken(t *testing.T) {
	svc := NewJWTService("test-secret", 1)

	_, err := svc.ValidateToken("not.a.token")

	if err == nil {
		t.Error("expected error for malformed token, got nil")
	}
}

func TestValidateToken_TamperedPayload(t *testing.T) {
	// Arrange — generate a valid token then replace the payload with forged claims
	svc := NewJWTService("test-secret", 1)
	u := testUser()
	tokenStr, _ := svc.GenerateToken(context.Background(), u)

	parts := strings.Split(tokenStr, ".")
	forgedPayload := fmt.Sprintf(
		`{"sub":"attacker","email":"attacker@example.com","role":"ADMIN","iat":%d,"exp":%d}`,
		time.Now().Unix(),
		time.Now().Add(time.Hour).Unix(),
	)
	parts[1] = base64.RawURLEncoding.EncodeToString([]byte(forgedPayload))
	tamperedToken := strings.Join(parts, ".")

	// Act
	_, err := svc.ValidateToken(tamperedToken)

	// Assert
	if err == nil {
		t.Error("expected error for tampered payload, got nil")
	}
}
