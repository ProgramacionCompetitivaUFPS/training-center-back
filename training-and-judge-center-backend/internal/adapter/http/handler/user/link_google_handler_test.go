package user

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newHandlerWithLinkGoogle(uc *appuser.LinkGoogleIdentityUseCase) *Handler {
	return &Handler{linkGoogle: uc}
}

func TestLinkGoogle_ValidToken_Returns204(t *testing.T) {
	const userID = "user-abc"

	oauthRepo := &mockHandlerOAuthIdentityRepo{}
	uc := appuser.NewLinkGoogleIdentityUseCase(&mockHandlerUserRepo{}, oauthRepo, &mockHandlerGoogleVerifier{}, &mockHandlerEmailSender{})
	h := newHandlerWithLinkGoogle(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.LinkGoogle),
		&domainuser.TokenClaims{UserID: userID, Role: shared.RoleContestant},
	)

	body := strings.NewReader(`{"id_token":"valid-token"}`)
	req := httptest.NewRequest(http.MethodPost, "/users/google", body)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestLinkGoogle_MissingIDToken_Returns400(t *testing.T) {
	uc := appuser.NewLinkGoogleIdentityUseCase(&mockHandlerUserRepo{}, &mockHandlerOAuthIdentityRepo{}, &mockHandlerGoogleVerifier{}, &mockHandlerEmailSender{})
	h := newHandlerWithLinkGoogle(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.LinkGoogle),
		&domainuser.TokenClaims{UserID: "user-abc", Role: shared.RoleContestant},
	)

	body := strings.NewReader(`{"id_token":""}`)
	req := httptest.NewRequest(http.MethodPost, "/users/google", body)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestLinkGoogle_AlreadyLinkedElsewhere_Returns409(t *testing.T) {
	oauthRepo := &mockHandlerOAuthIdentityRepo{
		saveFn: func(_ context.Context, _ *domainuser.OAuthIdentity) error {
			return apperror.NewConflict(domainuser.ErrCodeOAuthIdentityConflict, "this Google account is already linked to a user")
		},
	}
	uc := appuser.NewLinkGoogleIdentityUseCase(&mockHandlerUserRepo{}, oauthRepo, &mockHandlerGoogleVerifier{}, &mockHandlerEmailSender{})
	h := newHandlerWithLinkGoogle(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.LinkGoogle),
		&domainuser.TokenClaims{UserID: "user-abc", Role: shared.RoleContestant},
	)

	body := strings.NewReader(`{"id_token":"valid-token"}`)
	req := httptest.NewRequest(http.MethodPost, "/users/google", body)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d — body: %s", rr.Code, rr.Body.String())
	}
	var appErr apperror.AppError
	if err := json.NewDecoder(rr.Body).Decode(&appErr); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if appErr.Code != domainuser.ErrCodeOAuthIdentityConflict {
		t.Errorf("expected code %q, got %q", domainuser.ErrCodeOAuthIdentityConflict, appErr.Code)
	}
}

func TestLinkGoogle_InvalidJSON_Returns400(t *testing.T) {
	h := newHandlerWithLinkGoogle(nil)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.LinkGoogle),
		&domainuser.TokenClaims{UserID: "user-abc", Role: shared.RoleContestant},
	)

	req := httptest.NewRequest(http.MethodPost, "/users/google", strings.NewReader("not json"))
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
