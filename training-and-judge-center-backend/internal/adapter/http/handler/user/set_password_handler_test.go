package user

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newHandlerWithSetPassword(uc *appuser.SetPasswordUseCase) *Handler {
	return &Handler{setPassword: uc}
}

func googleOnlyUser(id string) *domainuser.User {
	return domainuser.RestoreUser(
		id,
		nil,
		"",
		"Google User",
		id,
		"",
		"",
		"",
		shared.RoleContestant.String(),
		domainuser.StatusActive.String(),
		time.Now(),
		nil,
		nil,
	)
}

func TestSetPassword_ValidPassword_Returns204(t *testing.T) {
	const userID = "user-abc"

	userRepo := &mockHandlerUserRepo{
		findByIDFn: func(_ context.Context, _ string) (*domainuser.User, error) {
			return googleOnlyUser(userID), nil
		},
	}
	uc := appuser.NewSetPasswordUseCase(userRepo, &mockHandlerEmailSender{})
	h := newHandlerWithSetPassword(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.SetPassword),
		&domainuser.TokenClaims{UserID: userID, Role: shared.RoleContestant},
	)

	body := strings.NewReader(`{"newPassword":"NewSecret1!"}`)
	req := httptest.NewRequest(http.MethodPost, "/users/password", body)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestSetPassword_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithSetPassword(nil)

	req := httptest.NewRequest(http.MethodPost, "/users/password", strings.NewReader(`{"newPassword":"NewSecret1!"}`))
	rr := httptest.NewRecorder()

	h.SetPassword(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestSetPassword_InvalidJSON_Returns400(t *testing.T) {
	h := newHandlerWithSetPassword(nil)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.SetPassword),
		&domainuser.TokenClaims{UserID: "user-abc", Role: shared.RoleContestant},
	)

	req := httptest.NewRequest(http.MethodPost, "/users/password", strings.NewReader("not json"))
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestSetPassword_AlreadyHasPassword_Returns409(t *testing.T) {
	const userID = "user-abc"

	userRepo := &mockHandlerUserRepo{
		findByIDFn: func(_ context.Context, _ string) (*domainuser.User, error) {
			return activeUserWithNoEmail(userID), nil
		},
	}
	uc := appuser.NewSetPasswordUseCase(userRepo, &mockHandlerEmailSender{})
	h := newHandlerWithSetPassword(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.SetPassword),
		&domainuser.TokenClaims{UserID: userID, Role: shared.RoleContestant},
	)

	body := strings.NewReader(`{"newPassword":"NewSecret1!"}`)
	req := httptest.NewRequest(http.MethodPost, "/users/password", body)
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
	if appErr.Code != appuser.ErrCodePasswordAlreadySet {
		t.Errorf("expected code %q, got %q", appuser.ErrCodePasswordAlreadySet, appErr.Code)
	}
}
