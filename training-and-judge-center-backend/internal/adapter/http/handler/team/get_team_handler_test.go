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
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newHandlerWithGetTeam(teamRepo domainTeam.Repository, memberRepo domainTeam.MemberRepository, invRepo domainTeam.InvitationRepository, userProvider appTeam.UserProvider) *Handler {
	return &Handler{getTeam: appTeam.NewGetTeamUseCase(teamRepo, memberRepo, invRepo, userProvider)}
}

func TestGetTeam_UnauthenticatedReturns401(t *testing.T) {
	h := newHandlerWithGetTeam(&mockTeamRepo{}, &mockMemberRepo{}, &mockInvitationRepo{}, &mockUserProvider{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/teams/t1", nil)
	wrapAuth(http.HandlerFunc(h.GetTeam)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetTeam_NotFoundReturns404(t *testing.T) {
	teamRepo := &mockTeamRepo{
		findByIDFn: func(_ string) (*domainTeam.Team, error) {
			return nil, apperror.NewNotFound(domainTeam.ErrCodeTeamNotFound, "team not found")
		},
	}
	h := newHandlerWithGetTeam(teamRepo, &mockMemberRepo{}, &mockInvitationRepo{}, &mockUserProvider{})
	w := httptest.NewRecorder()
	r := authedGetRequest("/teams/missing")
	wrapAuth(http.HandlerFunc(h.GetTeam)).ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetTeam_AnyAuthenticatedUserCanView(t *testing.T) {
	now := time.Now()
	uid := shared.RestoreUserID("u1")
	team := domainTeam.RestoreTeam("t1", domainTeam.RestoreTeamName("Alpha"), uid, now)
	member := domainTeam.RestoreTeamMember("m1", "t1", uid, now)

	teamRepo := &mockTeamRepo{
		findByIDFn: func(_ string) (*domainTeam.Team, error) { return team, nil },
	}
	memberRepo := &mockMemberRepo{
		findByTeamFn: func(_ string) ([]*domainTeam.TeamMember, error) {
			return []*domainTeam.TeamMember{member}, nil
		},
	}
	userProv := &mockUserProvider{
		displaysFn: func(ids []string) (map[string]*appTeam.UserDisplay, error) {
			m := make(map[string]*appTeam.UserDisplay)
			for _, id := range ids {
				m[id] = &appTeam.UserDisplay{Nickname: "nick_" + id}
			}
			return m, nil
		},
	}

	h := newHandlerWithGetTeam(teamRepo, memberRepo, &mockInvitationRepo{}, userProv)
	w := httptest.NewRecorder()
	r := authedGetRequest("/teams/t1")
	wrapAuth(http.HandlerFunc(h.GetTeam)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body getTeamResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body.ID != "t1" {
		t.Errorf("ID = %q, want t1", body.ID)
	}
	if body.Name != "Alpha" {
		t.Errorf("Name = %q, want Alpha", body.Name)
	}
	if len(body.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(body.Members))
	}
	if body.Members[0].Nickname != "nick_u1" {
		t.Errorf("Nickname = %q, want nick_u1", body.Members[0].Nickname)
	}
	if body.Members[0].JoinedAt == "" {
		t.Error("expected non-empty JoinedAt")
	}
}
