package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/domain/user"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func validClaims() *user.TokenClaims {
	email, _ := user.NewEmail("test@example.com")
	return &user.TokenClaims{
		UserID:   "user-123",
		Email:    email,
		Role:     shared.RoleContestant,
		IssuedAt: time.Now(),
	}
}

func alwaysValidTokenSvc() *mockTokenService {
	return &mockTokenService{
		validateTokenFn: func(_ string) (*user.TokenClaims, error) {
			return validClaims(), nil
		},
	}
}

func TestAuth_MissingAuthorizationHeader(t *testing.T) {
	// Arrange
	handler := Auth(alwaysValidTokenSvc(), nil)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rr, req)

	// Assert
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuth_MalformedBearerPrefix(t *testing.T) {
	// Arrange — "BearerXYZ" without the required space after "Bearer"
	handler := Auth(alwaysValidTokenSvc(), nil)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "BearerXYZ some-token")
	rr := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rr, req)

	// Assert
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	// Arrange
	tokenSvc := &mockTokenService{
		validateTokenFn: func(_ string) (*user.TokenClaims, error) {
			return nil, errors.New("invalid signature")
		},
	}
	handler := Auth(tokenSvc, nil)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer bad.token.here")
	rr := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rr, req)

	// Assert
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuth_RevokedSession(t *testing.T) {
	// Arrange
	sessionInv := &mockSessionInvalidator{
		isSessionRevokedFn: func(_ context.Context, _ string, _ time.Time) (bool, error) {
			return true, nil
		},
	}
	handler := Auth(alwaysValidTokenSvc(), sessionInv)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid.token.here")
	rr := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rr, req)

	// Assert
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuth_SessionInvalidatorError(t *testing.T) {
	// Arrange — IsSessionRevoked returns an error: Redis unavailable → 503
	sessionInv := &mockSessionInvalidator{
		isSessionRevokedFn: func(_ context.Context, _ string, _ time.Time) (bool, error) {
			return false, errors.New("redis timeout")
		},
	}
	handler := Auth(alwaysValidTokenSvc(), sessionInv)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid.token.here")
	rr := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rr, req)

	// Assert
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
}

func TestAuth_ValidToken_ClaimsInContext(t *testing.T) {
	// Arrange
	expected := validClaims()
	tokenSvc := &mockTokenService{
		validateTokenFn: func(_ string) (*user.TokenClaims, error) {
			return expected, nil
		},
	}
	var capturedClaims *user.TokenClaims
	capturingHandler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedClaims = GetClaims(r.Context())
	})
	handler := Auth(tokenSvc, nil)(capturingHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid.token.here")
	rr := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rr, req)

	// Assert
	if capturedClaims == nil {
		t.Fatal("expected claims in context, got nil")
	}
	if capturedClaims.UserID != expected.UserID {
		t.Errorf("expected UserID %q, got %q", expected.UserID, capturedClaims.UserID)
	}
	if capturedClaims.Role != expected.Role {
		t.Errorf("expected Role %q, got %q", expected.Role, capturedClaims.Role)
	}
	if capturedClaims.Email != expected.Email {
		t.Errorf("expected Email %v, got %v", expected.Email, capturedClaims.Email)
	}
	_ = rr
}

func TestRequireRole_CorrectRole(t *testing.T) {
	// Arrange
	handler := RequireRole(shared.RoleAdmin)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), claimsKey, &user.TokenClaims{
		UserID: "admin-123",
		Role:   shared.RoleAdmin,
	})
	rr := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rr, req.WithContext(ctx))

	// Assert
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestRequireRole_WrongRole(t *testing.T) {
	// Arrange
	handler := RequireRole(shared.RoleAdmin)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), claimsKey, &user.TokenClaims{
		UserID: "coach-123",
		Role:   shared.RoleCoach,
	})
	rr := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rr, req.WithContext(ctx))

	// Assert
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestRequireRole_NoClaims(t *testing.T) {
	// Arrange — no claims in context (request that bypassed Auth middleware)
	handler := RequireRole(shared.RoleAdmin)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rr, req)

	// Assert
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}
