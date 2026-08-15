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

func newHandlerWithLoginWithGoogle(uc *appuser.LoginWithGoogleUseCase) *AuthHandler {
	return &AuthHandler{loginWithGoogleUseCase: uc}
}

func newGoogleLoginUseCaseForHandlerTest(
	userRepo *mockUserRepo,
	oauthRepo *mockOAuthIdentityRepo,
	tokenSvc *mockTokenService,
	googleVerifier *mockGoogleVerifier,
) *appuser.LoginWithGoogleUseCase {
	return appuser.NewLoginWithGoogleUseCase(
		userRepo, oauthRepo, &mockRefreshTokenRepo{}, tokenSvc, &mockRefreshTokenCodec{}, googleVerifier, &mockTransactionManager{},
	)
}

func TestLoginWithGoogle_ValidToken_ReturnsSessionAndSetsCookie(t *testing.T) {
	activeUser := testActiveUser(t, "Str0ng!Pass")

	oauthRepo := &mockOAuthIdentityRepo{
		findByProviderFn: func(_ context.Context, _ domainuser.OAuthProvider, _ string) (*domainuser.OAuthIdentity, error) {
			identity, _ := domainuser.NewOAuthIdentity("identity-1", activeUser.ID(), domainuser.OAuthProviderGoogle, "google-sub-123", time.Now())
			return identity, nil
		},
	}
	userRepo := &mockUserRepo{
		findByIDFn: func(_ context.Context, id string) (*domainuser.User, error) {
			if id == activeUser.ID() {
				return activeUser, nil
			}
			return nil, nil
		},
	}
	tokenSvc := &mockTokenService{
		generateTokenFn: func(_ context.Context, _ *domainuser.User, _ string) (string, error) { return "new-access-token", nil },
	}
	googleVerifier := &mockGoogleVerifier{}

	uc := newGoogleLoginUseCaseForHandlerTest(userRepo, oauthRepo, tokenSvc, googleVerifier)
	h := newHandlerWithLoginWithGoogle(uc)

	body := strings.NewReader(`{"id_token":"valid-token","rememberSession":true}`)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/google", body)
	rec := httptest.NewRecorder()

	h.LoginWithGoogle(rec, req)

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

	var resp loginResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Token != "new-access-token" {
		t.Errorf("expected token %q, got %q", "new-access-token", resp.Token)
	}
	if resp.User.Email != "test@example.com" {
		t.Errorf("expected user email in response, got %q", resp.User.Email)
	}
}

func TestLoginWithGoogle_InvalidToken_Returns401(t *testing.T) {
	googleVerifier := &mockGoogleVerifier{
		verifyFn: func(_ context.Context, _ string) (*appuser.GoogleClaims, error) {
			return nil, apperror.NewUnauthorized("IGNORED", "google rejected the token")
		},
	}
	uc := newGoogleLoginUseCaseForHandlerTest(&mockUserRepo{}, &mockOAuthIdentityRepo{}, &mockTokenService{}, googleVerifier)
	h := newHandlerWithLoginWithGoogle(uc)

	body := strings.NewReader(`{"id_token":"bad-token"}`)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/google", body)
	rec := httptest.NewRecorder()

	h.LoginWithGoogle(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
	var appErr apperror.AppError
	if err := json.NewDecoder(rec.Body).Decode(&appErr); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if appErr.Code != appuser.ErrCodeInvalidGoogleToken {
		t.Errorf("expected code %q, got %q", appuser.ErrCodeInvalidGoogleToken, appErr.Code)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == refreshCookieName {
			t.Error("expected no refresh_token cookie to be set on a failed login")
		}
	}
}

func TestLoginWithGoogle_MissingIDToken_Returns400(t *testing.T) {
	uc := newGoogleLoginUseCaseForHandlerTest(&mockUserRepo{}, &mockOAuthIdentityRepo{}, &mockTokenService{}, &mockGoogleVerifier{})
	h := newHandlerWithLoginWithGoogle(uc)

	body := strings.NewReader(`{"id_token":""}`)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/google", body)
	rec := httptest.NewRecorder()

	h.LoginWithGoogle(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLoginWithGoogle_InvalidJSON(t *testing.T) {
	h := newHandlerWithLoginWithGoogle(nil)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/google", strings.NewReader("not json"))
	rec := httptest.NewRecorder()

	h.LoginWithGoogle(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
