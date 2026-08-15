package auth

import (
	"context"
	"testing"

	"google.golang.org/api/idtoken"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestGoogleVerifier_Verify_MapsClaims(t *testing.T) {
	v := &GoogleVerifier{
		clientID: "test-client-id",
		validate: func(_ context.Context, idToken, audience string) (*idtoken.Payload, error) {
			if idToken != "valid-token" {
				t.Errorf("expected idToken %q, got %q", "valid-token", idToken)
			}
			if audience != "test-client-id" {
				t.Errorf("expected audience %q, got %q", "test-client-id", audience)
			}
			return &idtoken.Payload{
				Subject: "google-sub-123",
				Claims: map[string]interface{}{
					"email":          "a@b.com",
					"email_verified": true,
					"name":           "A B",
				},
			}, nil
		},
	}

	claims, err := v.Verify(context.Background(), "valid-token")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if claims.Sub != "google-sub-123" {
		t.Errorf("expected Sub %q, got %q", "google-sub-123", claims.Sub)
	}
	if claims.Email != "a@b.com" {
		t.Errorf("expected Email %q, got %q", "a@b.com", claims.Email)
	}
	if !claims.EmailVerified {
		t.Error("expected EmailVerified true")
	}
	if claims.Name != "A B" {
		t.Errorf("expected Name %q, got %q", "A B", claims.Name)
	}
}

func TestGoogleVerifier_Verify_ValidationFails_ReturnsUnauthorized(t *testing.T) {
	v := &GoogleVerifier{
		clientID: "test-client-id",
		validate: func(_ context.Context, _, _ string) (*idtoken.Payload, error) {
			return nil, errValidationFailed
		},
	}

	_, err := v.Verify(context.Background(), "bad-token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != appuser.ErrCodeInvalidGoogleToken {
		t.Errorf("expected code %q, got %q", appuser.ErrCodeInvalidGoogleToken, appErr.Code)
	}
	if appErr.Kind != apperror.KindUnauthorized {
		t.Errorf("expected kind UNAUTHORIZED, got %s", appErr.Kind)
	}
}

func TestGoogleVerifier_Verify_MissingOptionalClaims_NoPanic(t *testing.T) {
	v := &GoogleVerifier{
		clientID: "test-client-id",
		validate: func(_ context.Context, _, _ string) (*idtoken.Payload, error) {
			return &idtoken.Payload{
				Subject: "google-sub-456",
				Claims:  map[string]interface{}{}, // no email, email_verified, or name
			}, nil
		},
	}

	claims, err := v.Verify(context.Background(), "token")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if claims.Email != "" {
		t.Errorf("expected empty Email, got %q", claims.Email)
	}
	if claims.EmailVerified {
		t.Error("expected EmailVerified false")
	}
	if claims.Name != "" {
		t.Errorf("expected empty Name, got %q", claims.Name)
	}
}

func TestNewGoogleVerifier_DefaultsToIdtokenValidate(t *testing.T) {
	v := NewGoogleVerifier("client-id")
	if v.clientID != "client-id" {
		t.Errorf("expected clientID %q, got %q", "client-id", v.clientID)
	}
	if v.validate == nil {
		t.Fatal("expected validate to default to idtoken.Validate, got nil")
	}
}

type validationError struct{ msg string }

func (e *validationError) Error() string { return e.msg }

var errValidationFailed = &validationError{msg: "idtoken: invalid token"}
