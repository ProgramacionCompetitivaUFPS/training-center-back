package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newHandlerWithRefresh(refreshUseCase *appuser.RefreshUseCase) *AuthHandler {
	return &AuthHandler{refreshUseCase: refreshUseCase}
}

func activeTestRefreshToken(now time.Time) *domainuser.RefreshToken {
	return domainuser.RestoreRefreshToken(
		"token-id", "user-id", "family-id", "old-hash",
		now, now.Add(domainuser.DefaultSessionCeiling),
		nil, nil, nil, nil,
	)
}

func testRequestWithRefreshCookie(secret string) *http.Request {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/refresh", nil)
	if secret != "" {
		req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: secret})
	}
	return req
}

func TestRefresh_Success_RotatesCookie(t *testing.T) {
	now := time.Now()
	refreshTokenRepo := &mockRefreshTokenRepo{
		findByTokenHashFn: func(_ context.Context, _ string) (*domainuser.RefreshToken, error) {
			return activeTestRefreshToken(now), nil
		},
		rotateFn: func(_ context.Context, _ string, _ *domainuser.RefreshToken) (bool, error) { return true, nil },
	}
	userRepo := &mockUserRepo{
		findByIDFn: func(_ context.Context, _ string) (*domainuser.User, error) {
			return testActiveUser(t, "Str0ng!Pass"), nil
		},
	}
	tokenSvc := &mockTokenService{
		generateTokenFn: func(_ context.Context, _ *domainuser.User) (string, error) { return "new-access-token", nil },
	}
	refreshUseCase := appuser.NewRefreshUseCase(refreshTokenRepo, userRepo, tokenSvc, &mockRefreshTokenCodec{}, &mockRateLimiter{}, &mockRotationCache{})
	h := newHandlerWithRefresh(refreshUseCase)

	req := testRequestWithRefreshCookie("raw-secret")
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var refreshCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == refreshCookieName {
			refreshCookie = c
		}
	}
	if refreshCookie == nil {
		t.Fatal("expected a rotated refresh_token cookie")
	}
	if refreshCookie.Value == "raw-secret" {
		t.Error("expected the cookie to carry a NEW secret, not the one presented")
	}

	var resp refreshResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Token != "new-access-token" {
		t.Errorf("expected token %q, got %q", "new-access-token", resp.Token)
	}
}

func TestRefresh_MissingCookie_RejectsWithoutCallingUseCase(t *testing.T) {
	executed := false
	refreshTokenRepo := &mockRefreshTokenRepo{
		findByTokenHashFn: func(_ context.Context, _ string) (*domainuser.RefreshToken, error) {
			executed = true
			return nil, nil
		},
	}
	refreshUseCase := appuser.NewRefreshUseCase(refreshTokenRepo, &mockUserRepo{}, &mockTokenService{}, &mockRefreshTokenCodec{}, &mockRateLimiter{}, &mockRotationCache{})
	h := newHandlerWithRefresh(refreshUseCase)

	req := testRequestWithRefreshCookie("")
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if executed {
		t.Error("expected the use case NOT to run when the refresh cookie is missing")
	}
}

func TestRefresh_UseCaseError_DoesNotSetCookie(t *testing.T) {
	refreshTokenRepo := &mockRefreshTokenRepo{
		findByTokenHashFn: func(_ context.Context, _ string) (*domainuser.RefreshToken, error) { return nil, nil },
	}
	refreshUseCase := appuser.NewRefreshUseCase(refreshTokenRepo, &mockUserRepo{}, &mockTokenService{}, &mockRefreshTokenCodec{}, &mockRateLimiter{}, &mockRotationCache{})
	h := newHandlerWithRefresh(refreshUseCase)

	req := testRequestWithRefreshCookie("raw-secret")
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
	var appErr apperror.AppError
	if err := json.NewDecoder(rec.Body).Decode(&appErr); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if appErr.Code != apperror.ErrCodeUnauthorized {
		t.Errorf("expected code %q, got %q", apperror.ErrCodeUnauthorized, appErr.Code)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == refreshCookieName {
			t.Error("expected no refresh_token cookie to be set on a failed refresh")
		}
	}
}

func TestRefresh_GraceWindowRace_DoesNotOverwriteCookie(t *testing.T) {
	now := time.Now()
	revokedAt := now.Add(-2 * time.Second) // within the grace window
	revoked := domainuser.RestoreRefreshToken(
		"token-id", "user-id", "family-id", "old-hash",
		now.Add(-time.Hour), now.Add(domainuser.DefaultSessionCeiling),
		&revokedAt, nil, nil, nil,
	)
	cachedOutput := &appuser.RefreshOutput{Token: "cached-access-token", RefreshToken: "", SessionExpiresAt: now}
	refreshTokenRepo := &mockRefreshTokenRepo{
		findByTokenHashFn: func(_ context.Context, _ string) (*domainuser.RefreshToken, error) { return revoked, nil },
	}
	userRepo := &mockUserRepo{
		findByIDFn: func(_ context.Context, _ string) (*domainuser.User, error) {
			return testActiveUser(t, "Str0ng!Pass"), nil
		},
	}
	cache := &mockRotationCache{
		getFn: func(_ context.Context, _ string) (*appuser.RefreshOutput, error) { return cachedOutput, nil },
	}
	refreshUseCase := appuser.NewRefreshUseCase(refreshTokenRepo, userRepo, &mockTokenService{}, &mockRefreshTokenCodec{}, &mockRateLimiter{}, cache)
	h := newHandlerWithRefresh(refreshUseCase)

	req := testRequestWithRefreshCookie("raw-secret")
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == refreshCookieName {
			t.Error("expected no new refresh_token cookie when the rotation cache returns an empty RefreshToken")
		}
	}
}
