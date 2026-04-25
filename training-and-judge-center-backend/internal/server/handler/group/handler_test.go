package group

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	domainUser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/internal/server/middleware"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// stubListGroupsUC is a no-op use case for tests that don't reach Execute.
type stubListGroupsUC struct{}

func (s *stubListGroupsUC) Execute(_ context.Context, _ appGroup.ListGroupsInput) (*appGroup.ListGroupsOutput, error) {
	return &appGroup.ListGroupsOutput{}, nil
}

type stubListMyGroupsUC struct{}

func (s *stubListMyGroupsUC) Execute(_ context.Context, _ appGroup.ListMyGroupsInput) (*appGroup.ListMyGroupsOutput, error) {
	return &appGroup.ListMyGroupsOutput{}, nil
}

// mockTokenSvc always validates successfully with a contestant role.
type mockTokenSvc struct{}

func (m *mockTokenSvc) GenerateToken(_ *domainUser.User) (string, error) { return "tok", nil }
func (m *mockTokenSvc) ValidateToken(_ string) (*domainUser.TokenClaims, error) {
	return &domainUser.TokenClaims{UserID: "u1", Role: domainUser.RoleContestant}, nil
}

func authedRequest(method, target string) *http.Request {
	r := httptest.NewRequest(method, target, nil)
	r.Header.Set("Authorization", "Bearer tok")
	return r
}

func wrapAuth(h http.Handler) http.Handler {
	return middleware.Auth(&mockTokenSvc{}, nil)(h)
}

func TestListGroups_NonIntegerPageReturns400(t *testing.T) {
	h := NewHandler(&stubListGroupsUC{}, nil, nil)
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ListGroups)).ServeHTTP(w, authedRequest("GET", "/groups?page=abc"))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %s", body.Code)
	}
}

func TestListGroups_NonIntegerLimitReturns400(t *testing.T) {
	h := NewHandler(&stubListGroupsUC{}, nil, nil)
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ListGroups)).ServeHTTP(w, authedRequest("GET", "/groups?limit=xyz"))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %s", body.Code)
	}
}

func TestListMyGroups_NonIntegerPageReturns400(t *testing.T) {
	h := NewHandler(nil, nil, &stubListMyGroupsUC{})
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ListMyGroups)).ServeHTTP(w, authedRequest("GET", "/users/me/groups?page=abc"))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
