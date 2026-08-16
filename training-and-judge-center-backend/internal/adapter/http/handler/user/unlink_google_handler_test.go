package user

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
)

func newHandlerWithUnlinkGoogle(uc *appuser.UnlinkGoogleIdentityUseCase) *Handler {
	return &Handler{unlinkGoogle: uc}
}

func TestUnlinkGoogle_LinkedAccount_Returns204(t *testing.T) {
	oauthRepo := &mockHandlerOAuthIdentityRepo{
		findByUserIDFn: func(_ context.Context, _ string) (*domainuser.OAuthIdentity, error) {
			identity, err := domainuser.NewOAuthIdentity("identity-1", "user-abc", domainuser.OAuthProviderGoogle, "google-sub-1", time.Now())
			if err != nil {
				t.Fatalf("unexpected error building test identity: %v", err)
			}
			return identity, nil
		},
	}
	uc := appuser.NewUnlinkGoogleIdentityUseCase(oauthRepo)
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
		findByUserIDFn: func(_ context.Context, _ string) (*domainuser.OAuthIdentity, error) {
			return nil, nil
		},
	}
	uc := appuser.NewUnlinkGoogleIdentityUseCase(oauthRepo)
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
