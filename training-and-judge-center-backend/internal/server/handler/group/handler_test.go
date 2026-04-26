package group

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainUser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/internal/server/middleware"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// stubGroupRepo is a no-op repository for tests that don't reach the use case.
type stubGroupRepo struct{}

func (s *stubGroupRepo) Save(_ context.Context, _ *domainGroup.Group) error        { return nil }
func (s *stubGroupRepo) FindByID(_ context.Context, _ string) (*domainGroup.Group, error) {
	return nil, nil
}
func (s *stubGroupRepo) ExistsByName(_ context.Context, _ domainGroup.GroupName) (bool, error) {
	return false, nil
}
func (s *stubGroupRepo) FindDefault(_ context.Context) (*domainGroup.Group, error) { return nil, nil }
func (s *stubGroupRepo) Delete(_ context.Context, _ string) error                  { return nil }
func (s *stubGroupRepo) List(_ context.Context, _ domainGroup.ListFilters) ([]*domainGroup.Group, int, error) {
	return nil, 0, nil
}

// stubMemberRepo is a no-op member repository for tests that don't reach the use case.
type stubMemberRepo struct{}

func (s *stubMemberRepo) Save(_ context.Context, _ *domainGroup.GroupMember) error { return nil }
func (s *stubMemberRepo) SaveAll(_ context.Context, _ []*domainGroup.GroupMember) error {
	return nil
}
func (s *stubMemberRepo) FindByGroupAndUser(_ context.Context, _ string, _ shared.UserID) (*domainGroup.GroupMember, error) {
	return nil, nil
}
func (s *stubMemberRepo) FindByGroup(_ context.Context, _ string, _ domainGroup.MemberFilters) ([]*domainGroup.GroupMember, int, error) {
	return nil, 0, nil
}
func (s *stubMemberRepo) Delete(_ context.Context, _ string, _ shared.UserID) error { return nil }
func (s *stubMemberRepo) CountLeads(_ context.Context, _ string) (int, error)       { return 0, nil }
func (s *stubMemberRepo) CountMembers(_ context.Context, _ string) (int, error)     { return 0, nil }
func (s *stubMemberRepo) ListLeads(_ context.Context, _ string) ([]*domainGroup.GroupMember, error) {
	return nil, nil
}
func (s *stubMemberRepo) BulkStats(_ context.Context, _ []string, _ shared.UserID) (map[string]domainGroup.MemberStats, error) {
	return nil, nil
}

// stubUserProvider is a no-op user provider for tests that don't reach the use case.
type stubUserProvider struct{}

func (s *stubUserProvider) GetDisplays(_ context.Context, _ []string) (map[string]*appGroup.UserDisplay, error) {
	return nil, nil
}

// stubPrefsReader is a no-op preferences reader for tests that don't reach the use case.
type stubPrefsReader struct{}

func (s *stubPrefsReader) HideGlobalGroup(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func stubHandler() *Handler {
	repo := &stubGroupRepo{}
	memberRepo := &stubMemberRepo{}
	return NewHandler(
		appGroup.NewListGroupsUseCase(repo, memberRepo),
		appGroup.NewGetGroupUseCase(repo, memberRepo, &stubUserProvider{}),
		appGroup.NewListMyGroupsUseCase(repo, memberRepo, &stubPrefsReader{}),
	)
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
	h := stubHandler()
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
	h := stubHandler()
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
	h := stubHandler()
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ListMyGroups)).ServeHTTP(w, authedRequest("GET", "/users/me/groups?page=abc"))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
