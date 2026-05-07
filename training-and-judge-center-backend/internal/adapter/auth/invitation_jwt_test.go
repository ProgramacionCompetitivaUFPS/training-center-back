package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestGenerateInviteToken_ClaimsPresent(t *testing.T) {
	svc := NewGroupInvitationJWTService("test-secret")

	tokenStr, err := svc.GenerateInviteToken("g-123", "u-456")
	if err != nil {
		t.Fatalf("GenerateInviteToken returned unexpected error: %v", err)
	}

	token, err := jwt.ParseWithClaims(tokenStr, &invitationJWTClaims{}, func(t *jwt.Token) (any, error) {
		return []byte("test-secret"), nil
	})
	if err != nil {
		t.Fatalf("could not parse generated token: %v", err)
	}
	claims := token.Claims.(*invitationJWTClaims)
	if claims.GroupID != "g-123" {
		t.Errorf("GroupID: got %q, want %q", claims.GroupID, "g-123")
	}
	if claims.InviterID != "u-456" {
		t.Errorf("InviterID: got %q, want %q", claims.InviterID, "u-456")
	}
	if claims.Issuer != invitationIssuer {
		t.Errorf("Issuer: got %q, want %q", claims.Issuer, invitationIssuer)
	}
	if claims.ExpiresAt == nil {
		t.Error("exp claim is missing")
	}
}

func TestValidateInviteToken_ValidToken(t *testing.T) {
	svc := NewGroupInvitationJWTService("test-secret")
	tokenStr, err := svc.GenerateInviteToken("g-123", "u-456")
	if err != nil {
		t.Fatalf("GenerateInviteToken returned unexpected error: %v", err)
	}

	claims, err := svc.ValidateInviteToken(tokenStr)

	if err != nil {
		t.Fatalf("ValidateInviteToken returned unexpected error: %v", err)
	}
	if claims.GroupID != "g-123" {
		t.Errorf("GroupID: got %q, want %q", claims.GroupID, "g-123")
	}
}

func TestValidateInviteToken_ExpiredToken(t *testing.T) {
	// Craft an already-expired token by back-dating IssuedAt and ExpiresAt.
	secret := []byte("test-secret")
	expired := invitationJWTClaims{
		GroupID:   "g-1",
		InviterID: "u-1",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    invitationIssuer,
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-80 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-8 * time.Hour)),
		},
	}
	tokenStr, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, expired).SignedString(secret)

	svc := NewGroupInvitationJWTService("test-secret")
	_, err := svc.ValidateInviteToken(tokenStr)

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != appGroup.ErrCodeExpiredInviteToken {
		t.Fatalf("expected EXPIRED_INVITE_TOKEN, got %v", err)
	}
}

func TestValidateInviteToken_WrongKey(t *testing.T) {
	generator := NewGroupInvitationJWTService("original-secret")
	validator := NewGroupInvitationJWTService("different-secret")
	tokenStr, _ := generator.GenerateInviteToken("g-1", "u-1")

	_, err := validator.ValidateInviteToken(tokenStr)

	if err == nil {
		t.Error("expected error for token signed with a different key, got nil")
	}
}

func TestValidateInviteToken_AlgorithmNone(t *testing.T) {
	svc := NewGroupInvitationJWTService("test-secret")
	noneToken := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"group_id":   "g-1",
		"inviter_id": "attacker",
		"iss":        invitationIssuer,
		"iat":        time.Now().Unix(),
		"exp":        time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, err := noneToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("could not sign none token: %v", err)
	}

	_, err = svc.ValidateInviteToken(tokenStr)

	if err == nil {
		t.Error("expected error for alg:none token (algorithm confusion attack), got nil")
	}
}

func TestValidateInviteToken_MalformedToken(t *testing.T) {
	svc := NewGroupInvitationJWTService("test-secret")

	_, err := svc.ValidateInviteToken("not.a.token")

	if err == nil {
		t.Error("expected error for malformed token, got nil")
	}
}

func TestValidateInviteToken_SessionTokenRejected(t *testing.T) {
	// A login token signed with the same secret but no Issuer must be rejected.
	secret := []byte("shared-secret")
	sessionToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   "u-1",
		"email": "user@example.com",
		"role":  "CONTESTANT",
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, err := sessionToken.SignedString(secret)
	if err != nil {
		t.Fatalf("could not sign session token: %v", err)
	}

	svc := NewGroupInvitationJWTService("shared-secret")
	_, err = svc.ValidateInviteToken(tokenStr)

	if err == nil {
		t.Error("expected error: session token must not be accepted as invite token")
	}
}
