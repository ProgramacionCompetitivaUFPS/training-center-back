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
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
)

// newHandlerWithResetPassword builds a Handler with only resetPassword wired.
func newHandlerWithResetPassword(uc *appuser.ResetPasswordUseCase) *Handler {
	return &Handler{resetPassword: uc}
}

// pendingRecoveryRequest returns a valid pending PasswordRecoveryRequest for userID with the given code.
func pendingRecoveryRequest(userID, code string) *domainuser.PasswordRecoveryRequest {
	return domainuser.RestorePasswordRecoveryRequest(
		"req-001",
		userID,
		code,
		domainuser.RequestStatusPending,
		time.Now().Add(time.Hour),
		time.Now(),
		nil,
	)
}

// TestResetPassword_GuardRejectsInvalidInputs_Returns400 verifies that the composite guard
// rejects requests with empty email, invalid code, or empty newPassword before calling the use case.
func TestResetPassword_GuardRejectsInvalidInputs_Returns400(t *testing.T) {
	cases := []struct {
		name        string
		email       string
		code        string
		newPassword string
	}{
		{"empty email", "", "123456", "NewSecret2@"},
		{"invalid code (letters)", "user@example.com", "abc123", "NewSecret2@"},
		{"invalid code (5 digits)", "user@example.com", "12345", "NewSecret2@"},
		{"empty newPassword", "user@example.com", "123456", ""},
	}

	uc := appuser.NewResetPasswordUseCase(nil, nil, nil, nil)
	h := newHandlerWithResetPassword(uc)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := strings.NewReader(strings.Join([]string{
				`{"email":`, jsonString(tc.email),
				`,"code":`, jsonString(tc.code),
				`,"newPassword":`, jsonString(tc.newPassword), `}`,
			}, ""))
			req := httptest.NewRequest(http.MethodPost, "/users/password/reset", body)
			rr := httptest.NewRecorder()

			http.HandlerFunc(h.ResetPassword).ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("%s: expected 400, got %d", tc.name, rr.Code)
			}
		})
	}
}

// TestResetPassword_SessionsNotInvalidated_Returns200WithCode verifies that when the use
// case returns ErrSessionsNotInvalidated, the handler responds with 200 and the special code.
func TestResetPassword_SessionsNotInvalidated_Returns200WithCode(t *testing.T) {
	const userEmail = "user@example.com"
	const userID = "user-abc"
	const validCode = "123456"

	userRepo := &mockHandlerUserRepo{
		findByEmailFn: func(_ context.Context, _ domainuser.Email) (*domainuser.User, error) {
			return activeUserWithNoEmail(userID), nil
		},
	}
	recoveryRepo := &mockHandlerPasswordRecoveryRepo{
		findPendingByUserIDFn: func(_ context.Context, _ string) (*domainuser.PasswordRecoveryRequest, error) {
			return pendingRecoveryRequest(userID, validCode), nil
		},
	}
	sessionInvalidator := &mockHandlerSessionInvalidator{
		invalidateAllUserSessionsFn: func(_ context.Context, _ string, _ time.Time) error {
			return errors.New("redis down")
		},
	}
	uc := appuser.NewResetPasswordUseCase(
		userRepo,
		recoveryRepo,
		sessionInvalidator,
		&mockHandlerTxManager{},
	)
	h := newHandlerWithResetPassword(uc)

	body := strings.NewReader(`{"email":"user@example.com","code":"123456","newPassword":"NewSecret2@"}`)
	req := httptest.NewRequest(http.MethodPost, "/users/password/reset", body)
	rr := httptest.NewRecorder()

	http.HandlerFunc(h.ResetPassword).ServeHTTP(rr, req)

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

// TestResetPassword_Success_Returns200WithMessage verifies that when the use case returns
// nil, the handler responds with 200 and the success message.
func TestResetPassword_Success_Returns200WithMessage(t *testing.T) {
	const userID = "user-abc"
	const validCode = "123456"

	userRepo := &mockHandlerUserRepo{
		findByEmailFn: func(_ context.Context, _ domainuser.Email) (*domainuser.User, error) {
			return activeUserWithNoEmail(userID), nil
		},
	}
	recoveryRepo := &mockHandlerPasswordRecoveryRepo{
		findPendingByUserIDFn: func(_ context.Context, _ string) (*domainuser.PasswordRecoveryRequest, error) {
			return pendingRecoveryRequest(userID, validCode), nil
		},
	}
	uc := appuser.NewResetPasswordUseCase(
		userRepo,
		recoveryRepo,
		&mockHandlerSessionInvalidator{},
		&mockHandlerTxManager{},
	)
	h := newHandlerWithResetPassword(uc)

	body := strings.NewReader(`{"email":"user@example.com","code":"123456","newPassword":"NewSecret2@"}`)
	req := httptest.NewRequest(http.MethodPost, "/users/password/reset", body)
	rr := httptest.NewRecorder()

	http.HandlerFunc(h.ResetPassword).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if resp["message"] == "" {
		t.Error("expected non-empty success message in response")
	}
}

// jsonString returns the JSON encoding of s (a quoted string).
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
