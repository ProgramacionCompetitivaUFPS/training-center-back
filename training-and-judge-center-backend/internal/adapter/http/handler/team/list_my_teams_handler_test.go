package team

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appTeam "github.com/training-judge-center/backend/internal/application/team"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

func authedGetRequest(target string) *http.Request {
	r := httptest.NewRequest("GET", target, nil)
	r.Header.Set("Authorization", "Bearer tok")
	return r
}

func newHandlerWithListMyTeams(teamRepo domainTeam.Repository, memberRepo domainTeam.MemberRepository) *Handler {
	return &Handler{listMyTeams: appTeam.NewListMyTeamsUseCase(teamRepo, memberRepo)}
}

func TestListMyTeams_UnauthenticatedReturns401(t *testing.T) {
	h := newHandlerWithListMyTeams(&mockTeamRepo{}, &mockMemberRepo{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/users/me/teams", nil)
	wrapAuth(http.HandlerFunc(h.ListMyTeams)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListMyTeams_ReturnsTeams(t *testing.T) {
	now := time.Now()
	uid := shared.RestoreUserID("u1")
	team := domainTeam.RestoreTeam("t1", domainTeam.RestoreTeamName("Alpha"), uid, now)
	member := domainTeam.RestoreTeamMember("m1", "t1", uid, now)

	teamRepo := &mockTeamRepo{
		findByIDsFn: func(_ []string) ([]*domainTeam.Team, error) {
			return []*domainTeam.Team{team}, nil
		},
	}
	memberRepo := &mockMemberRepo{
		findByUserFn: func(_ shared.UserID) ([]*domainTeam.TeamMember, error) {
			return []*domainTeam.TeamMember{member}, nil
		},
		bulkCountFn: func(_ []string) (map[string]int, error) {
			return map[string]int{"t1": 2}, nil
		},
	}

	h := newHandlerWithListMyTeams(teamRepo, memberRepo)
	w := httptest.NewRecorder()
	r := authedGetRequest("/users/me/teams")
	wrapAuth(http.HandlerFunc(h.ListMyTeams)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body listMyTeamsResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if len(body.Teams) != 1 {
		t.Fatalf("expected 1 team, got %d", len(body.Teams))
	}
	if body.Teams[0].ID != "t1" {
		t.Errorf("ID = %q, want t1", body.Teams[0].ID)
	}
	if body.Teams[0].MemberCount != 2 {
		t.Errorf("MemberCount = %d, want 2", body.Teams[0].MemberCount)
	}
}

func TestListMyTeams_EmptyListReturns200(t *testing.T) {
	h := newHandlerWithListMyTeams(&mockTeamRepo{}, &mockMemberRepo{})
	w := httptest.NewRecorder()
	r := authedGetRequest("/users/me/teams")
	wrapAuth(http.HandlerFunc(h.ListMyTeams)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body listMyTeamsResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if len(body.Teams) != 0 {
		t.Errorf("expected 0 teams, got %d", len(body.Teams))
	}
}
