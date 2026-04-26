package user

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appuser "github.com/training-judge-center/backend/internal/application/user"
)

// newHandlerWithRequestPasswordRecovery builds a UserHandler with only requestPasswordRecovery wired.
func newHandlerWithRequestPasswordRecovery(uc *appuser.RequestPasswordRecoveryUseCase) *UserHandler {
	return &UserHandler{requestPasswordRecovery: uc}
}

// TestRequestPasswordRecovery_EmptyEmail_Returns400 verifies that the guard rejects
// a request with no email before the use case is called.
func TestRequestPasswordRecovery_EmptyEmail_Returns400(t *testing.T) {
	uc := appuser.NewRequestPasswordRecoveryUseCase(nil, nil, nil, nil)
	h := newHandlerWithRequestPasswordRecovery(uc)

	body := strings.NewReader(`{"email":""}`)
	req := httptest.NewRequest(http.MethodPost, "/users/password/recovery", body)
	rr := httptest.NewRecorder()

	http.HandlerFunc(h.RequestPasswordRecovery).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

// TestRequestPasswordRecovery_WithEmail_Returns200 verifies that a non-empty email passes
// the guard and the use case executes, returning 200 (ambiguous response regardless of
// whether the user exists).
func TestRequestPasswordRecovery_WithEmail_Returns200(t *testing.T) {
	// userRepo.FindByEmail returns nil (user not found) — UC returns nil immediately,
	// which is the intentional ambiguous response path.
	uc := appuser.NewRequestPasswordRecoveryUseCase(
		&mockHandlerUserRepo{},
		&mockHandlerPasswordRecoveryRepo{},
		&mockHandlerEmailSender{},
		&mockHandlerRateLimiter{},
	)
	h := newHandlerWithRequestPasswordRecovery(uc)

	body := strings.NewReader(`{"email":"user@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/users/password/recovery", body)
	rr := httptest.NewRecorder()

	http.HandlerFunc(h.RequestPasswordRecovery).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}
}
