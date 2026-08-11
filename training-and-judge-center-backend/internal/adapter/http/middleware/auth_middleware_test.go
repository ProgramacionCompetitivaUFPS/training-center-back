package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
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
	handler := Auth(alwaysValidTokenSvc(), &mockSessionInvalidator{})(okHandler())
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
	handler := Auth(alwaysValidTokenSvc(), &mockSessionInvalidator{})(okHandler())
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
	handler := Auth(tokenSvc, &mockSessionInvalidator{})(okHandler())
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
		isAllUserSessionRevokedFn: func(_ context.Context, _ string, _ time.Time) (bool, error) {
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
	// Arrange — IsAllUserSessionRevoked returns an error: Redis unavailable → 503
	sessionInv := &mockSessionInvalidator{
		isAllUserSessionRevokedFn: func(_ context.Context, _ string, _ time.Time) (bool, error) {
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

func TestAuth_InvalidatedSpecificSession(t *testing.T) {
	// Arrange — user-level check passes, but this specific session was invalidated
	claims := validClaims()
	claims.SessionID = "family-123"
	tokenSvc := &mockTokenService{
		validateTokenFn: func(_ string) (*user.TokenClaims, error) { return claims, nil },
	}
	sessionInv := &mockSessionInvalidator{
		isSessionInvalidatedFn: func(_ context.Context, sessionID string, _ time.Time) (bool, error) {
			return sessionID == "family-123", nil
		},
	}
	handler := Auth(tokenSvc, sessionInv)(okHandler())
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

func TestAuth_SpecificSessionInvalidatorError(t *testing.T) {
	// Arrange — IsSessionInvalidated returns an error: Redis unavailable → 503
	claims := validClaims()
	claims.SessionID = "family-123"
	tokenSvc := &mockTokenService{
		validateTokenFn: func(_ string) (*user.TokenClaims, error) { return claims, nil },
	}
	sessionInv := &mockSessionInvalidator{
		isSessionInvalidatedFn: func(_ context.Context, _ string, _ time.Time) (bool, error) {
			return false, errors.New("redis timeout")
		},
	}
	handler := Auth(tokenSvc, sessionInv)(okHandler())
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

func TestAuth_EmptySessionID_SkipsSessionCheck(t *testing.T) {
	// Arrange — token issued before the sid claim existed (SessionID == ""); the
	// per-session check must be skipped rather than querying Redis with an empty key.
	claims := validClaims()
	claims.SessionID = ""
	tokenSvc := &mockTokenService{
		validateTokenFn: func(_ string) (*user.TokenClaims, error) { return claims, nil },
	}
	sessionInv := &mockSessionInvalidator{
		isSessionInvalidatedFn: func(_ context.Context, _ string, _ time.Time) (bool, error) {
			t.Fatal("IsSessionInvalidated should not be called when SessionID is empty")
			return false, nil
		},
	}
	handler := Auth(tokenSvc, sessionInv)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid.token.here")
	rr := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rr, req)

	// Assert
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestAuth_ValidToken_CurrentUserInContext(t *testing.T) {
	// Arrange
	expected := validClaims()
	tokenSvc := &mockTokenService{
		validateTokenFn: func(_ string) (*user.TokenClaims, error) {
			return expected, nil
		},
	}
	var capturedUser appshared.CurrentUser
	var capturedOk bool
	capturingHandler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedUser, capturedOk = GetCurrentUser(r.Context())
	})
	handler := Auth(tokenSvc, &mockSessionInvalidator{})(capturingHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid.token.here")
	rr := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rr, req)

	// Assert
	if !capturedOk {
		t.Fatal("expected current user in context, got none")
	}
	if capturedUser.ID != expected.UserID {
		t.Errorf("expected ID %q, got %q", expected.UserID, capturedUser.ID)
	}
	if capturedUser.Role != expected.Role {
		t.Errorf("expected Role %q, got %q", expected.Role, capturedUser.Role)
	}
	_ = rr
}

func TestRequireRole_CorrectRole(t *testing.T) {
	// Arrange
	handler := RequireRole(shared.RoleAdmin)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), currentUserKey, appshared.CurrentUser{
		ID:   "admin-123",
		Role: shared.RoleAdmin,
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
	ctx := context.WithValue(req.Context(), currentUserKey, appshared.CurrentUser{
		ID:   "coach-123",
		Role: shared.RoleCoach,
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
	// Arrange — no current user in context (request that bypassed Auth middleware)
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
