package user

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
)

func newHandlerWithUnlinkGoogle(uc *appuser.UnlinkGoogleIdentityUseCase) *Handler {
	return &Handler{unlinkGoogle: uc}
}

func TestUnlinkGoogle_LinkedAccount_Returns204(t *testing.T) {
	oauthRepo := &mockHandlerOAuthIdentityRepo{
		deleteByUserIDFn: func(_ context.Context, _ string, _ domainuser.OAuthProvider) (bool, error) {
			return true, nil
		},
	}
	userRepo := &mockHandlerUserRepo{
		findByIDFn: func(_ context.Context, _ string) (*domainuser.User, error) {
			return activeUserWithNoEmail("user-abc"), nil
		},
	}
	uc := appuser.NewUnlinkGoogleIdentityUseCase(userRepo, oauthRepo, &mockHandlerEmailSender{})
	h := newHandlerWithUnlinkGoogle(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.UnlinkGoogle),
		&domainuser.TokenClaims{UserID: "user-abc", Role: shared.RoleContestant},
	)

	req := httptest.NewRequest(http.MethodDelete, "/users/google", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestUnlinkGoogle_NoLinkedAccount_Returns404(t *testing.T) {
	oauthRepo := &mockHandlerOAuthIdentityRepo{
		deleteByUserIDFn: func(_ context.Context, _ string, _ domainuser.OAuthProvider) (bool, error) {
			return false, nil
		},
	}
	userRepo := &mockHandlerUserRepo{
		findByIDFn: func(_ context.Context, _ string) (*domainuser.User, error) {
			return activeUserWithNoEmail("user-abc"), nil
		},
	}
	uc := appuser.NewUnlinkGoogleIdentityUseCase(userRepo, oauthRepo, &mockHandlerEmailSender{})
	h := newHandlerWithUnlinkGoogle(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.UnlinkGoogle),
		&domainuser.TokenClaims{UserID: "user-abc", Role: shared.RoleContestant},
	)

	req := httptest.NewRequest(http.MethodDelete, "/users/google", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestUnlinkGoogle_NoPasswordSet_Returns409(t *testing.T) {
	userRepo := &mockHandlerUserRepo{
		findByIDFn: func(_ context.Context, _ string) (*domainuser.User, error) {
			return googleOnlyUser("user-abc"), nil
		},
	}
	uc := appuser.NewUnlinkGoogleIdentityUseCase(userRepo, &mockHandlerOAuthIdentityRepo{}, &mockHandlerEmailSender{})
	h := newHandlerWithUnlinkGoogle(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.UnlinkGoogle),
		&domainuser.TokenClaims{UserID: "user-abc", Role: shared.RoleContestant},
	)

	req := httptest.NewRequest(http.MethodDelete, "/users/google", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d — body: %s", rr.Code, rr.Body.String())
	}
}
