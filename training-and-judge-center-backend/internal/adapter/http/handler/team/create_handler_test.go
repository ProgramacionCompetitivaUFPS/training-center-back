package team

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	domainUser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ── stubs ────────────────────────────────────────────────────────────────────

type stubTeamRepo struct {
	existsByNameFn func(name domainTeam.TeamName) (bool, error)
	saveErr        error
}

func (s *stubTeamRepo) Save(_ context.Context, _ *domainTeam.Team) error { return s.saveErr }
func (s *stubTeamRepo) FindByID(_ context.Context, _ string) (*domainTeam.Team, error) {
	return nil, apperror.NewNotFound(domainTeam.ErrCodeTeamNotFound, "team not found")
}
func (s *stubTeamRepo) ExistsByName(_ context.Context, name domainTeam.TeamName) (bool, error) {
	if s.existsByNameFn != nil {
		return s.existsByNameFn(name)
	}
	return false, nil
}

type stubMemberRepo struct{}

func (s *stubMemberRepo) Save(_ context.Context, _ *domainTeam.TeamMember) error {
	return nil
}
func (s *stubMemberRepo) FindByTeam(_ context.Context, _ string) ([]*domainTeam.TeamMember, error) {
	return nil, nil
}

type stubUserProvider struct{}

func (s *stubUserProvider) GetDisplay(_ context.Context, _ string) (*appTeam.UserDisplay, error) {
	return &appTeam.UserDisplay{Nickname: "testuser"}, nil
}

type stubTxManager struct{}

func (s *stubTxManager) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type mockTokenSvc struct{}

func (m *mockTokenSvc) GenerateToken(_ context.Context, _ *domainUser.User) (string, error) {
	return "tok", nil
}
func (m *mockTokenSvc) ValidateToken(_ string) (*domainUser.TokenClaims, error) {
	return &domainUser.TokenClaims{UserID: "u1", Role: shared.RoleContestant}, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func stubHandler() *Handler {
	return NewHandler(appTeam.NewCreateTeamUseCase(
		&stubTeamRepo{},
		&stubMemberRepo{},
		&stubUserProvider{},
		&stubTxManager{},
	))
}

func wrapAuth(h http.Handler) http.Handler {
	return middleware.Auth(&mockTokenSvc{}, nil)(h)
}

func authedPostRequest(target, body string) *http.Request {
	r := httptest.NewRequest("POST", target, strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer tok")
	r.Header.Set("Content-Type", "application/json")
	return r
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestCreate_UnauthenticatedReturns401(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/teams", nil)
	wrapAuth(http.HandlerFunc(h.Create)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestCreate_InvalidJSONReturns400(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()
	r := authedPostRequest("/teams", `{invalid json}`)
	wrapAuth(http.HandlerFunc(h.Create)).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreate_EmptyNameReturns400(t *testing.T) {
	h := stubHandler()
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
	h := stubHandler()
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
	repo := &stubTeamRepo{
		existsByNameFn: func(_ domainTeam.TeamName) (bool, error) { return true, nil },
	}
	h := NewHandler(appTeam.NewCreateTeamUseCase(repo, &stubMemberRepo{}, &stubUserProvider{}, &stubTxManager{}))
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
	h := stubHandler()
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
