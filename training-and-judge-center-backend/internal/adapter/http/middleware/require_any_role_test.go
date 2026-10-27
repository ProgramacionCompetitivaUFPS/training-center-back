package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

func withCurrentUser(role shared.Role) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), currentUserKey, appshared.CurrentUser{ID: "user-123", Role: role})
	return req.WithContext(ctx)
}

func TestRequireAnyRole_AllowsMatchingRole(t *testing.T) {
	handler := RequireAnyRole(shared.RoleCoach, shared.RoleAdmin)(okHandler())
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, withCurrentUser(shared.RoleCoach))

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestRequireAnyRole_AllowsOtherMatchingRole(t *testing.T) {
	handler := RequireAnyRole(shared.RoleCoach, shared.RoleAdmin)(okHandler())
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, withCurrentUser(shared.RoleAdmin))

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestRequireAnyRole_RejectsNonMatchingRole(t *testing.T) {
	handler := RequireAnyRole(shared.RoleCoach, shared.RoleAdmin)(okHandler())
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, withCurrentUser(shared.RoleContestant))

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestRequireAnyRole_RejectsMissingCurrentUser(t *testing.T) {
	handler := RequireAnyRole(shared.RoleCoach, shared.RoleAdmin)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}
