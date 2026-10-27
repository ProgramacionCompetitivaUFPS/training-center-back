package user

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
)

func newHandlerWithGetMyProfile(uc *appuser.GetMyProfileUseCase) *Handler {
	return &Handler{getMyProfile: uc}
}

func TestGetMyProfile_Unauthenticated_Returns401(t *testing.T) {
	uc := appuser.NewGetMyProfileUseCase(&mockHandlerUserRepo{}, &mockHandlerOAuthIdentityRepo{})
	h := newHandlerWithGetMyProfile(uc)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	rr := httptest.NewRecorder()

	h.GetMyProfile(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestGetMyProfile_GoogleLinked_ReturnsTrue(t *testing.T) {
	const userID = "user-abc"

	userRepo := &mockHandlerUserRepo{
		findByIDFn: func(_ context.Context, _ string) (*domainuser.User, error) {
			return activeUserWithNoEmail(userID), nil
		},
	}
	oauthRepo := &mockHandlerOAuthIdentityRepo{
		findByUserIDFn: func(_ context.Context, userID string, _ domainuser.OAuthProvider) (*domainuser.OAuthIdentity, error) {
			return domainuser.RestoreOAuthIdentity("identity-1", userID, domainuser.OAuthProviderGoogle, "google-sub-1", time.Now()), nil
		},
	}
	uc := appuser.NewGetMyProfileUseCase(userRepo, oauthRepo)
	h := newHandlerWithGetMyProfile(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.GetMyProfile),
		&domainuser.TokenClaims{UserID: userID, Role: shared.RoleContestant},
	)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}
	var resp fullUserResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !resp.GoogleLinked {
		t.Error("expected googleLinked to be true in response")
	}
}

func TestGetMyProfile_GoogleNotLinked_ReturnsFalse(t *testing.T) {
	const userID = "user-abc"

	userRepo := &mockHandlerUserRepo{
		findByIDFn: func(_ context.Context, _ string) (*domainuser.User, error) {
			return activeUserWithNoEmail(userID), nil
		},
	}
	uc := appuser.NewGetMyProfileUseCase(userRepo, &mockHandlerOAuthIdentityRepo{})
	h := newHandlerWithGetMyProfile(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.GetMyProfile),
		&domainuser.TokenClaims{UserID: userID, Role: shared.RoleContestant},
	)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}
	var resp fullUserResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.GoogleLinked {
		t.Error("expected googleLinked to be false in response")
	}
}

func TestGetMyProfile_HasPassword_ReturnsTrue(t *testing.T) {
	const userID = "user-abc"

	userRepo := &mockHandlerUserRepo{
		findByIDFn: func(_ context.Context, _ string) (*domainuser.User, error) {
			return activeUserWithNoEmail(userID), nil
		},
	}
	uc := appuser.NewGetMyProfileUseCase(userRepo, &mockHandlerOAuthIdentityRepo{})
	h := newHandlerWithGetMyProfile(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.GetMyProfile),
		&domainuser.TokenClaims{UserID: userID, Role: shared.RoleContestant},
	)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}
	var resp fullUserResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !resp.HasPassword {
		t.Error("expected hasPassword to be true in response")
	}
}

func TestGetMyProfile_NoPassword_ReturnsFalse(t *testing.T) {
	const userID = "user-abc"

	googleOnlyUser := domainuser.RestoreUser(
		userID,
		nil,
		"",
		"Google User",
		userID,
		"",
		"",
		"",
		shared.RoleContestant.String(),
		domainuser.StatusActive.String(),
		time.Now(),
		nil,
		nil,
	)
	userRepo := &mockHandlerUserRepo{
		findByIDFn: func(_ context.Context, _ string) (*domainuser.User, error) {
			return googleOnlyUser, nil
		},
	}
	uc := appuser.NewGetMyProfileUseCase(userRepo, &mockHandlerOAuthIdentityRepo{})
	h := newHandlerWithGetMyProfile(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.GetMyProfile),
		&domainuser.TokenClaims{UserID: userID, Role: shared.RoleContestant},
	)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}
	var resp fullUserResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.HasPassword {
		t.Error("expected hasPassword to be false in response")
	}
}
