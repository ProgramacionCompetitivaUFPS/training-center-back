package user

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
)

// newHandlerWithConfirmEmailChange builds a UserHandler with only confirmEmailChange wired.
func newHandlerWithConfirmEmailChange(uc *appuser.ConfirmEmailChangeUseCase) *UserHandler {
	return &UserHandler{confirmEmailChange: uc}
}

// newHandlerWithRequestEmailChange builds a UserHandler with only requestEmailChange wired.
func newHandlerWithRequestEmailChange(uc *appuser.RequestEmailChangeUseCase) *UserHandler {
	return &UserHandler{requestEmailChange: uc}
}

// activeUserWithEmail returns a valid ACTIVE user with the given email address.
func activeUserWithEmail(id, email string) *domainuser.User {
	p, _ := domainuser.NewPassword("Secret1!")
	return domainuser.RestoreUser(
		id,
		&email,
		p.Hash(),
		"Test User",
		"testuser"+id,
		"Country",
		"City",
		"Institution",
		shared.RoleContestant.String(),
		domainuser.StatusActive.String(),
		time.Now(),
		nil,
		nil,
	)
}

// pendingEmailChangeRequest returns a valid pending EmailChangeRequest for userID with the given code.
func pendingEmailChangeRequest(id, userID, code string) *domainuser.EmailChangeRequest {
	newEmail, _ := domainuser.NewEmail("new@example.com")
	return domainuser.RestoreEmailChangeRequest(
		id,
		userID,
		newEmail,
		code,
		domainuser.StatusPending,
		time.Now().Add(time.Hour),
		time.Now(),
		nil,
	)
}

// TestConfirmEmailChange_InvalidCodes_Return400 verifies that the digitCodeRegex guard
// rejects any code that is not exactly 6 numeric digits before the use case is called.
func TestConfirmEmailChange_InvalidCodes_Return400(t *testing.T) {
	cases := []struct {
		name string
		code string
	}{
		{"letters mixed in", "abc123"},
		{"too short (5 digits)", "12345"},
		{"too long (7 digits)", "1234567"},
	}

	uc := appuser.NewConfirmEmailChangeUseCase(nil, nil, nil, nil)
	h := newHandlerWithConfirmEmailChange(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.ConfirmEmailChange),
		&domainuser.TokenClaims{UserID: "user-abc", Role: shared.RoleContestant},
	)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := strings.NewReader(fmt.Sprintf(`{"code":%q}`, tc.code))
			req := httptest.NewRequest(http.MethodPost, "/users/me/email/confirm", body)
			req.Header.Set("Authorization", "Bearer test-token")
			rr := httptest.NewRecorder()

			wrapped.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("code %q: expected 400, got %d", tc.code, rr.Code)
			}
		})
	}
}

// TestConfirmEmailChange_ValidSixDigitCode_Returns200 verifies that a valid 6-digit code
// passes the guard, reaches the use case, and the handler responds with 200.
func TestConfirmEmailChange_ValidSixDigitCode_Returns200(t *testing.T) {
	const userID = "user-abc"
	const validCode = "654321"

	userRepo := &mockHandlerUserRepo{
		findByIDFn: func(_ context.Context, _ string) (*domainuser.User, error) {
			return activeUserWithEmail(userID, "old@example.com"), nil
		},
	}
	emailChangeRepo := &mockHandlerEmailChangeRepo{
		findByCodeAndUserIDFn: func(_ context.Context, _, _ string) (*domainuser.EmailChangeRequest, error) {
			return pendingEmailChangeRequest("req-001", userID, validCode), nil
		},
	}
	uc := appuser.NewConfirmEmailChangeUseCase(
		userRepo,
		emailChangeRepo,
		&mockHandlerEmailSender{},
		&mockHandlerTxManager{},
	)
	h := newHandlerWithConfirmEmailChange(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.ConfirmEmailChange),
		&domainuser.TokenClaims{UserID: userID, Role: shared.RoleContestant},
	)

	body := strings.NewReader(fmt.Sprintf(`{"code":%q}`, validCode))
	req := httptest.NewRequest(http.MethodPost, "/users/me/email/confirm", body)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

// TestRequestEmailChange_MissingFields_Return400 verifies that the guard rejects
// requests with empty password or empty newEmail before the use case is called.
func TestRequestEmailChange_MissingFields_Return400(t *testing.T) {
	cases := []struct {
		name     string
		password string
		newEmail string
	}{
		{"empty password", "", "new@example.com"},
		{"empty newEmail", "Secret1!", ""},
	}

	uc := appuser.NewRequestEmailChangeUseCase(nil, nil, nil, nil)
	h := newHandlerWithRequestEmailChange(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.RequestEmailChange),
		&domainuser.TokenClaims{UserID: "user-abc", Role: shared.RoleContestant},
	)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := strings.NewReader(fmt.Sprintf(`{"password":%q,"newEmail":%q}`, tc.password, tc.newEmail))
			req := httptest.NewRequest(http.MethodPost, "/users/me/email/request", body)
			req.Header.Set("Authorization", "Bearer test-token")
			rr := httptest.NewRecorder()

			wrapped.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("%s: expected 400, got %d", tc.name, rr.Code)
			}
		})
	}
}

// TestRequestEmailChange_ValidFields_PassesToUseCase verifies that valid password and
// newEmail pass the guard and the use case executes successfully, returning 200.
func TestRequestEmailChange_ValidFields_PassesToUseCase(t *testing.T) {
	const userID = "user-abc"

	userRepo := &mockHandlerUserRepo{
		findByIDFn: func(_ context.Context, _ string) (*domainuser.User, error) {
			return activeUserWithNoEmail(userID), nil
		},
	}
	uc := appuser.NewRequestEmailChangeUseCase(
		userRepo,
		&mockHandlerEmailChangeRepo{},
		&mockHandlerEmailSender{},
		&mockHandlerRateLimiter{},
	)
	h := newHandlerWithRequestEmailChange(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.RequestEmailChange),
		&domainuser.TokenClaims{UserID: userID, Role: shared.RoleContestant},
	)

	body := strings.NewReader(`{"password":"Secret1!","newEmail":"new@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/users/me/email/request", body)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}
}
