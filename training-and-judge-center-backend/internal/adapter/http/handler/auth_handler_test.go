package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newHandlerWithLogin(loginUseCase *appuser.LoginUseCase) *AuthHandler {
	return &AuthHandler{loginUseCase: loginUseCase}
}

func testActiveUser(t *testing.T, plainPassword string) *domainuser.User {
	t.Helper()
	email, err := domainuser.NewEmail("test@example.com")
	if err != nil {
		t.Fatalf("failed to build test email: %v", err)
	}
	password, err := domainuser.NewPassword(plainPassword)
	if err != nil {
		t.Fatalf("failed to build test password: %v", err)
	}
	nickname, err := domainuser.NewNickname("testuser")
	if err != nil {
		t.Fatalf("failed to build test nickname: %v", err)
	}
	u, err := domainuser.NewUser("user-id", time.Now(), email, password, "Test User", nickname, "Colombia", "Cucuta", "UFPS")
	if err != nil {
		t.Fatalf("failed to build test user: %v", err)
	}
	return u
}

func TestLogin_Success_SetsRefreshCookieAndReturnsSessionExpiresAt(t *testing.T) {
	const plainPassword = "Str0ng!Pass"
	testUser := testActiveUser(t, plainPassword)

	userRepo := &mockUserRepo{
		findByEmailFn: func(_ context.Context, _ domainuser.Email) (*domainuser.User, error) {
			return testUser, nil
		},
	}
	refreshTokenRepo := &mockRefreshTokenRepo{}
	tokenSvc := &mockTokenService{
		generateTokenFn: func(_ context.Context, _ *domainuser.User) (string, error) { return "new-access-token", nil },
	}
	loginUseCase := appuser.NewLoginUseCase(userRepo, refreshTokenRepo, tokenSvc, &mockRefreshTokenCodec{}, &mockRateLimiter{})
	h := newHandlerWithLogin(loginUseCase)

	body := strings.NewReader(`{"email":"test@example.com","password":"` + plainPassword + `","rememberSession":true}`)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/login", body)
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	cookies := rec.Result().Cookies()
	var refreshCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == refreshCookieName {
			refreshCookie = c
		}
	}
	if refreshCookie == nil {
		t.Fatal("expected refresh_token cookie to be set")
	}
	if !refreshCookie.HttpOnly || !refreshCookie.Secure || refreshCookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("expected HttpOnly+Secure+SameSite=Strict cookie, got %+v", refreshCookie)
	}
	if refreshCookie.Path != refreshCookiePath {
		t.Errorf("expected cookie path %q, got %q", refreshCookiePath, refreshCookie.Path)
	}

	var resp loginResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Token != "new-access-token" {
		t.Errorf("expected token %q, got %q", "new-access-token", resp.Token)
	}
	if resp.SessionExpiresAt == "" {
		t.Error("expected sessionExpiresAt to be set")
	}
	if resp.User.Email != "test@example.com" {
		t.Errorf("expected user email in response, got %q", resp.User.Email)
	}
}

func TestLogin_InvalidJSON(t *testing.T) {
	h := newHandlerWithLogin(nil)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/login", strings.NewReader("not json"))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestLogin_UseCaseError_PropagatesWithoutSettingCookie(t *testing.T) {
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ context.Context, _ domainuser.Email) (*domainuser.User, error) { return nil, nil },
	}
	loginUseCase := appuser.NewLoginUseCase(userRepo, &mockRefreshTokenRepo{}, &mockTokenService{}, &mockRefreshTokenCodec{}, &mockRateLimiter{})
	h := newHandlerWithLogin(loginUseCase)

	body := strings.NewReader(`{"email":"nobody@example.com","password":"Str0ng!Pass"}`)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/login", body)
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
	var appErr apperror.AppError
	if err := json.NewDecoder(rec.Body).Decode(&appErr); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if appErr.Code != appuser.ErrCodeInvalidCredentials {
		t.Errorf("expected code %q, got %q", appuser.ErrCodeInvalidCredentials, appErr.Code)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == refreshCookieName {
			t.Error("expected no refresh_token cookie to be set on a failed login")
		}
	}
}
