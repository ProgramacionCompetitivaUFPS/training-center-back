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
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/internal/server/middleware"
)

// newHandlerWithConfirmDeactivation builds a UserHandler with only confirmDeactivation
// wired. Other use cases are nil — if a test accidentally reaches them, Go will panic,
// which is the right signal that the test is exercising the wrong path.
func newHandlerWithConfirmDeactivation(uc *appuser.ConfirmDeactivationUseCase) *UserHandler {
	return &UserHandler{confirmDeactivation: uc}
}

// wrapWithAuth wraps h with the Auth middleware configured to always inject claims.
func wrapWithAuth(h http.Handler, claims *domainuser.TokenClaims) http.Handler {
	tokenSvc := &mockTokenService{
		validateFn: func(_ string) (*domainuser.TokenClaims, error) {
			return claims, nil
		},
	}
	return middleware.Auth(tokenSvc, nil)(h)
}

// activeUserWithNoEmail returns a valid ACTIVE user without an email field.
// The absence of email ensures the use case skips the confirmation email send,
// keeping mock dependencies minimal.
func activeUserWithNoEmail(id string) *domainuser.User {
	p, _ := domainuser.NewPassword("Secret1!")
	u, err := domainuser.RestoreUser(
		id,
		nil,
		p.Hash(),
		"Test User",
		"testuser",
		"Country",
		"City",
		"Institution",
		domainuser.RoleContestant.String(),
		domainuser.StatusActive.String(),
		time.Now(),
		nil,
		nil,
	)
	if err != nil {
		panic("activeUserWithNoEmail: " + err.Error())
	}
	return u
}

// pendingDeactRequest returns a valid pending deactivation request for userID with the given code.
func pendingDeactRequest(userID, code string) *domainuser.DeactivationRequest {
	return domainuser.RestoreDeactivationRequest(
		"req-001",
		userID,
		code,
		time.Now().Add(time.Hour),
		0,
		nil,
		domainuser.DeactivationStatusPending,
		time.Now(),
		time.Now(),
	)
}

// TestConfirmDeactivation_InvalidCodes_Return400 verifies that the digitCodeRegex
// guard in ConfirmDeactivation rejects any code that is not exactly 6 numeric digits
// before the use case is ever called.
func TestConfirmDeactivation_InvalidCodes_Return400(t *testing.T) {
	cases := []struct {
		name string
		code string
	}{
		{"letters mixed in", "abc123"},
		{"too short (5 digits)", "12345"},
		{"too long (7 digits)", "1234567"},
	}

	// Arrange: use case with nil repos — it must never be called for these inputs.
	uc := appuser.NewConfirmDeactivationUseCase(nil, nil, nil, nil, nil, nil)
	h := newHandlerWithConfirmDeactivation(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.ConfirmDeactivation),
		&domainuser.TokenClaims{UserID: "user-abc", Role: domainuser.RoleContestant},
	)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			body := strings.NewReader(fmt.Sprintf(`{"code":%q}`, tc.code))
			req := httptest.NewRequest(http.MethodPost, "/users/me/deactivation/confirm", body)
			req.Header.Set("Authorization", "Bearer test-token")
			rr := httptest.NewRecorder()

			// Act
			wrapped.ServeHTTP(rr, req)

			// Assert
			if rr.Code != http.StatusBadRequest {
				t.Errorf("code %q: expected 400, got %d", tc.code, rr.Code)
			}
		})
	}
}

// TestConfirmDeactivation_ValidSixDigitCode_Returns204 verifies that a code of exactly
// 6 numeric digits passes the guard and reaches the use case, which processes the
// deactivation and responds with 204 No Content.
func TestConfirmDeactivation_ValidSixDigitCode_Returns204(t *testing.T) {
	const userID = "user-abc"
	const validCode = "123456"

	// Arrange
	userRepo := &mockHandlerUserRepo{
		findByIDFn: func(_ context.Context, _ string) (*domainuser.User, error) {
			return activeUserWithNoEmail(userID), nil
		},
	}
	deactRepo := &mockHandlerDeactRepo{
		findPendingByUserIDFn: func(_ context.Context, _ string) (*domainuser.DeactivationRequest, error) {
			return pendingDeactRequest(userID, validCode), nil
		},
	}
	uc := appuser.NewConfirmDeactivationUseCase(
		userRepo,
		deactRepo,
		&mockHandlerDeactAuditRepo{},
		&mockHandlerEmailSender{},
		&mockHandlerSessionInvalidator{},
		&mockHandlerTxManager{},
	)
	h := newHandlerWithConfirmDeactivation(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.ConfirmDeactivation),
		&domainuser.TokenClaims{UserID: userID, Role: domainuser.RoleContestant},
	)

	body := strings.NewReader(fmt.Sprintf(`{"code":%q}`, validCode))
	req := httptest.NewRequest(http.MethodPost, "/users/me/deactivation/confirm", body)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	// Act
	wrapped.ServeHTTP(rr, req)

	// Assert
	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d — body: %s", rr.Code, rr.Body.String())
	}
}
