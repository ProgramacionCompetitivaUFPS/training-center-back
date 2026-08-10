package user

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
)

// newHandlerWithUpdatePassword builds a Handler with only updatePassword wired.
func newHandlerWithUpdatePassword(uc *appuser.UpdatePasswordUseCase) *Handler {
	return &Handler{updatePassword: uc}
}

// TestUpdatePassword_Success_Returns204 verifies that when the use case returns nil,
// the handler responds with 204 No Content.
func TestUpdatePassword_Success_Returns204(t *testing.T) {
	const userID = "user-abc"

	userRepo := &mockHandlerUserRepo{
		findByIDFn: func(_ context.Context, _ string) (*domainuser.User, error) {
			return activeUserWithEmail(userID, "user@example.com"), nil
		},
	}
	uc := appuser.NewUpdatePasswordUseCase(
		userRepo,
		&mockHandlerEmailSender{},
		&mockHandlerSessionInvalidator{},
		&mockHandlerRateLimiter{},
		&mockHandlerRefreshTokenRepo{},
	)
	h := newHandlerWithUpdatePassword(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.UpdatePassword),
		&domainuser.TokenClaims{UserID: userID, Role: shared.RoleContestant},
	)

	body := strings.NewReader(`{"currentPassword":"Secret1!","newPassword":"NewSecret2@"}`)
	req := httptest.NewRequest(http.MethodPut, "/users/me/password", body)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

// TestUpdatePassword_SessionsNotInvalidated_Returns200WithCode verifies that when the use
// case returns ErrSessionsNotInvalidated, the handler responds with 200 and the special code.
func TestUpdatePassword_SessionsNotInvalidated_Returns200WithCode(t *testing.T) {
	const userID = "user-abc"

	userRepo := &mockHandlerUserRepo{
		findByIDFn: func(_ context.Context, _ string) (*domainuser.User, error) {
			return activeUserWithEmail(userID, "user@example.com"), nil
		},
	}
	sessionInvalidator := &mockHandlerSessionInvalidator{
		invalidateAllUserSessionsFn: func(_ context.Context, _ string, _ time.Time) error {
			return errors.New("redis down")
		},
	}
	uc := appuser.NewUpdatePasswordUseCase(
		userRepo,
		&mockHandlerEmailSender{},
		sessionInvalidator,
		&mockHandlerRateLimiter{},
		&mockHandlerRefreshTokenRepo{},
	)
	h := newHandlerWithUpdatePassword(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.UpdatePassword),
		&domainuser.TokenClaims{UserID: userID, Role: shared.RoleContestant},
	)

	body := strings.NewReader(`{"currentPassword":"Secret1!","newPassword":"NewSecret2@"}`)
	req := httptest.NewRequest(http.MethodPut, "/users/me/password", body)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if resp["code"] != "SESSIONS_NOT_INVALIDATED" {
		t.Errorf("expected code SESSIONS_NOT_INVALIDATED, got %q", resp["code"])
	}
}
