package team

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	domainUser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ── mocks ─────────────────────────────────────────────────────────────────────

type mockTeamRepo struct {
	saveErr        error
	existsByNameFn func(name domainTeam.TeamName) (bool, error)
	findByIDFn     func(id string) (*domainTeam.Team, error)
	findByIDsFn    func(ids []string) ([]*domainTeam.Team, error)
}

func (m *mockTeamRepo) Save(_ context.Context, _ *domainTeam.Team) error { return m.saveErr }
func (m *mockTeamRepo) FindByID(_ context.Context, id string) (*domainTeam.Team, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id)
	}
	return nil, apperror.NewNotFound(domainTeam.ErrCodeTeamNotFound, "team not found")
}
func (m *mockTeamRepo) FindByIDs(_ context.Context, ids []string) ([]*domainTeam.Team, error) {
	if m.findByIDsFn != nil {
		return m.findByIDsFn(ids)
	}
	return []*domainTeam.Team{}, nil
}
func (m *mockTeamRepo) ExistsByName(_ context.Context, name domainTeam.TeamName) (bool, error) {
	if m.existsByNameFn != nil {
		return m.existsByNameFn(name)
	}
	return false, nil
}

type mockMemberRepo struct {
	saveErr               error
	findByTeamFn          func(teamID string) ([]*domainTeam.TeamMember, error)
	findByTeamAndUserFn   func(teamID string, userID shared.UserID) (*domainTeam.TeamMember, error)
	findByUserFn          func(userID shared.UserID) ([]*domainTeam.TeamMember, error)
	bulkCountFn           func(teamIDs []string) (map[string]int, error)
	deleteByTeamAndUserFn func(teamID string, userID shared.UserID) error
}

func (m *mockMemberRepo) Save(_ context.Context, _ *domainTeam.TeamMember) error { return m.saveErr }
func (m *mockMemberRepo) FindByTeam(_ context.Context, teamID string) ([]*domainTeam.TeamMember, error) {
	if m.findByTeamFn != nil {
		return m.findByTeamFn(teamID)
	}
	return []*domainTeam.TeamMember{}, nil
}
func (m *mockMemberRepo) FindByTeamAndUser(_ context.Context, teamID string, userID shared.UserID) (*domainTeam.TeamMember, error) {
	if m.findByTeamAndUserFn != nil {
		return m.findByTeamAndUserFn(teamID, userID)
	}
	return nil, apperror.NewNotFound(domainTeam.ErrCodeNotTeamMember, "not a member")
}
func (m *mockMemberRepo) FindByUser(_ context.Context, userID shared.UserID) ([]*domainTeam.TeamMember, error) {
	if m.findByUserFn != nil {
		return m.findByUserFn(userID)
	}
	return []*domainTeam.TeamMember{}, nil
}
func (m *mockMemberRepo) BulkCountByTeams(_ context.Context, teamIDs []string) (map[string]int, error) {
	if m.bulkCountFn != nil {
		return m.bulkCountFn(teamIDs)
	}
	return map[string]int{}, nil
}
func (m *mockMemberRepo) DeleteByTeamAndUser(_ context.Context, teamID string, userID shared.UserID) error {
	if m.deleteByTeamAndUserFn != nil {
		return m.deleteByTeamAndUserFn(teamID, userID)
	}
	return nil
}

type mockUserProvider struct {
	displayFn        func(userID string) (*appTeam.UserDisplay, error)
	displaysFn       func(userIDs []string) (map[string]*appTeam.UserDisplay, error)
	findByNicknameFn func(nickname string) (*appTeam.UserDisplay, error)
}

func (m *mockUserProvider) GetDisplay(_ context.Context, userID string) (*appTeam.UserDisplay, error) {
	if m.displayFn != nil {
		return m.displayFn(userID)
	}
	return &appTeam.UserDisplay{Nickname: "testuser"}, nil
}
func (m *mockUserProvider) GetDisplays(_ context.Context, userIDs []string) (map[string]*appTeam.UserDisplay, error) {
	if m.displaysFn != nil {
		return m.displaysFn(userIDs)
	}
	result := make(map[string]*appTeam.UserDisplay, len(userIDs))
	for _, id := range userIDs {
		result[id] = &appTeam.UserDisplay{ID: id, Nickname: "testuser"}
	}
	return result, nil
}
func (m *mockUserProvider) FindByNickname(_ context.Context, nickname string) (*appTeam.UserDisplay, error) {
	if m.findByNicknameFn != nil {
		return m.findByNicknameFn(nickname)
	}
	return &appTeam.UserDisplay{ID: "found-user-id", Nickname: nickname}, nil
}

type mockTxManager struct{}

func (m *mockTxManager) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type mockTokenSvc struct{}

func (m *mockTokenSvc) GenerateToken(_ context.Context, _ *domainUser.User, _ string) (string, error) {
	return "tok", nil
}
func (m *mockTokenSvc) ValidateToken(_ string) (*domainUser.TokenClaims, error) {
	return &domainUser.TokenClaims{UserID: "u1", Role: shared.RoleContestant}, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newHandlerWithCreate(teamRepo domainTeam.Repository, memberRepo domainTeam.MemberRepository, userProvider appTeam.UserProvider, txManager appshared.TransactionManager) *Handler {
	return &Handler{createTeam: appTeam.NewCreateTeamUseCase(teamRepo, memberRepo, userProvider, txManager)}
}

func wrapAuth(h http.Handler) http.Handler {
	return middleware.Auth(&mockTokenSvc{}, noopSessionInvalidator{})(h)
}

func authedPostRequest(target, body string) *http.Request {
	r := httptest.NewRequest("POST", target, strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer tok")
	r.Header.Set("Content-Type", "application/json")
	return r
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestCreate_UnauthenticatedReturns401(t *testing.T) {
	h := newHandlerWithCreate(&mockTeamRepo{}, &mockMemberRepo{}, &mockUserProvider{}, &mockTxManager{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/teams", nil)
	wrapAuth(http.HandlerFunc(h.Create)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestCreate_InvalidJSONReturns400(t *testing.T) {
	h := newHandlerWithCreate(&mockTeamRepo{}, &mockMemberRepo{}, &mockUserProvider{}, &mockTxManager{})
	w := httptest.NewRecorder()
	r := authedPostRequest("/teams", `{invalid json}`)
	wrapAuth(http.HandlerFunc(h.Create)).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreate_EmptyNameReturns400(t *testing.T) {
	h := newHandlerWithCreate(&mockTeamRepo{}, &mockMemberRepo{}, &mockUserProvider{}, &mockTxManager{})
	w := httptest.NewRecorder()
	r := authedPostRequest("/teams", `{"name":""}`)
	wrapAuth(http.HandlerFunc(h.Create)).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %s", body.Code)
	}
}

func TestCreate_ValidRequestReturns201(t *testing.T) {
	h := newHandlerWithCreate(&mockTeamRepo{}, &mockMemberRepo{}, &mockUserProvider{}, &mockTxManager{})
	w := httptest.NewRecorder()
	r := authedPostRequest("/teams", `{"name":"Alpha Team"}`)
	wrapAuth(http.HandlerFunc(h.Create)).ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body createTeamResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body.Name != "Alpha Team" {
		t.Errorf("Name = %q, want %q", body.Name, "Alpha Team")
	}
	if body.ID == "" {
		t.Error("expected non-empty ID")
	}
	if len(body.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(body.Members))
	}
	if body.Members[0].Nickname != "testuser" {
		t.Errorf("Nickname = %q, want %q", body.Members[0].Nickname, "testuser")
	}
}

func TestCreate_DuplicateNameReturns409(t *testing.T) {
	repo := &mockTeamRepo{
		existsByNameFn: func(_ domainTeam.TeamName) (bool, error) { return true, nil },
	}
	h := newHandlerWithCreate(repo, &mockMemberRepo{}, &mockUserProvider{}, &mockTxManager{})
	w := httptest.NewRecorder()
	r := authedPostRequest("/teams", `{"name":"Alpha Team"}`)
	wrapAuth(http.HandlerFunc(h.Create)).ServeHTTP(w, r)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body.Code != domainTeam.ErrCodeTeamNameExists {
		t.Errorf("expected TEAM_NAME_EXISTS, got %s", body.Code)
	}
}

func TestCreate_ResponseContainsJoinedAt(t *testing.T) {
	h := newHandlerWithCreate(&mockTeamRepo{}, &mockMemberRepo{}, &mockUserProvider{}, &mockTxManager{})
	w := httptest.NewRecorder()
	r := authedPostRequest("/teams", `{"name":"Beta Team"}`)
	wrapAuth(http.HandlerFunc(h.Create)).ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var body createTeamResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body.Members[0].JoinedAt == "" {
		t.Error("expected non-empty JoinedAt")
	}
}

// noopSessionInvalidator is a package-local no-op user.SessionInvalidator so
// handler tests do not depend on the sibling adapter/auth package.
type noopSessionInvalidator struct{}

func (noopSessionInvalidator) InvalidateAllUserSessions(context.Context, string, time.Time) error {
	return nil
}
func (noopSessionInvalidator) IsAllUserSessionRevoked(context.Context, string, time.Time) (bool, error) {
	return false, nil
}
func (noopSessionInvalidator) InvalidateSession(context.Context, string, time.Time) error {
	return nil
}
func (noopSessionInvalidator) IsSessionInvalidated(context.Context, string) (bool, error) {
	return false, nil
}
