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

func newHandlerWithLogout(logoutUseCase *appuser.LogoutUseCase) *AuthHandler {
	return &AuthHandler{logoutUseCase: logoutUseCase}
}

func testRequestWithLogoutCookie(wrapped string) *http.Request {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/logout", nil)
	if wrapped != "" {
		req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: wrapped})
	}
	return req
}

func assertClearedRefreshCookie(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	var refreshCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == refreshCookieName {
			refreshCookie = c
		}
	}
	if refreshCookie == nil {
		t.Fatal("expected a refresh_token cookie in the response (clearing it)")
	}
	if refreshCookie.MaxAge != -1 {
		t.Errorf("expected MaxAge -1 to clear the cookie, got %d", refreshCookie.MaxAge)
	}
}

func TestLogout_Success_ClearsCookieAndReturns204(t *testing.T) {
	now := time.Now()
	refreshTokenRepo := &mockRefreshTokenRepo{
		findByTokenHashFn: func(_ context.Context, _ string) (*domainuser.RefreshToken, error) {
			return activeTestRefreshToken(now), nil
		},
	}
	logoutUseCase := appuser.NewLogoutUseCase(refreshTokenRepo, &mockSessionInvalidator{}, &mockRefreshTokenCodec{})
	h := newHandlerWithLogout(logoutUseCase)

	req := testRequestWithLogoutCookie("wrapped-value")
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	assertClearedRefreshCookie(t, rec)
}

func TestLogout_NoCookie_Returns204AndClearsCookie(t *testing.T) {
	refreshTokenRepo := &mockRefreshTokenRepo{
		findByTokenHashFn: func(_ context.Context, _ string) (*domainuser.RefreshToken, error) {
			t.Fatal("FindByTokenHash should not be called when there is no cookie")
			return nil, nil
		},
	}
	logoutUseCase := appuser.NewLogoutUseCase(refreshTokenRepo, &mockSessionInvalidator{}, &mockRefreshTokenCodec{})
	h := newHandlerWithLogout(logoutUseCase)

	req := testRequestWithLogoutCookie("")
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	assertClearedRefreshCookie(t, rec)
}

func TestLogout_UseCaseError_PropagatesWithoutClearingCookie(t *testing.T) {
	refreshTokenRepo := &mockRefreshTokenRepo{
		findByTokenHashFn: func(_ context.Context, _ string) (*domainuser.RefreshToken, error) {
			return nil, apperror.NewInternal()
		},
	}
	logoutUseCase := appuser.NewLogoutUseCase(refreshTokenRepo, &mockSessionInvalidator{}, &mockRefreshTokenCodec{})
	h := newHandlerWithLogout(logoutUseCase)

	req := testRequestWithLogoutCookie("wrapped-value")
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}
	var appErr apperror.AppError
	if err := json.NewDecoder(rec.Body).Decode(&appErr); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == refreshCookieName {
			t.Error("expected no refresh_token cookie mutation on a failed logout")
		}
	}
}
