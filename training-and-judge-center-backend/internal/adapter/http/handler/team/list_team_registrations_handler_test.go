package team

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appTeam "github.com/training-judge-center/backend/internal/application/team"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newHandlerWithListTeamRegistrations(
	contestProvider appTeam.ContestProvider,
	groupChecker appTeam.GroupMemberChecker,
	participRepo domainContest.TeamParticipationRepository,
	teamRepo domainTeam.Repository,
	userProvider appTeam.UserProvider,
) *Handler {
	return &Handler{
		listTeamRegistrations: appTeam.NewListTeamRegistrationsUseCase(
			contestProvider, groupChecker, participRepo, teamRepo, userProvider,
		),
	}
}

func listRegistrationsRequest(groupID, contestID string) *http.Request {
	r := httptest.NewRequest("GET", "/groups/"+groupID+"/contests/"+contestID+"/team-registrations", nil)
	r.Header.Set("Authorization", "Bearer tok")
	r.SetPathValue("groupId", groupID)
	r.SetPathValue("contestId", contestID)
	return r
}

func TestListTeamRegistrations_UnauthenticatedReturns401(t *testing.T) {
	h := newHandlerWithListTeamRegistrations(
		&mockContestProvider{}, &mockGroupChecker{}, &mockTeamParticipRepo{},
		&mockTeamRepo{}, &mockUserProvider{},
	)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/groups/g1/contests/c1/team-registrations", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	wrapAuth(http.HandlerFunc(h.ListTeamRegistrations)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListTeamRegistrations_NotGroupMemberReturns403(t *testing.T) {
	groupChecker := &mockGroupChecker{
		fn: func(_ string, _ []string) (map[string]bool, error) {
			return map[string]bool{"u1": false}, nil
		},
	}
	h := newHandlerWithListTeamRegistrations(
		&mockContestProvider{}, groupChecker, &mockTeamParticipRepo{},
		&mockTeamRepo{}, &mockUserProvider{},
	)
	w := httptest.NewRecorder()
	r := listRegistrationsRequest("g1", "c1")
	wrapAuth(http.HandlerFunc(h.ListTeamRegistrations)).ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.Code != appTeam.ErrCodeMemberNotInGroup {
		t.Errorf("expected %s, got %s", appTeam.ErrCodeMemberNotInGroup, body.Code)
	}
}

func TestListTeamRegistrations_SuccessReturns200(t *testing.T) {
	now := time.Now()
	participRepo := &mockTeamParticipRepo{
		listFn: func(_ string, _, _ int) ([]*domainContest.ContestTeamParticipation, int, error) {
			p := domainContest.RestoreContestTeamParticipation("p1", "c1", "t1", []string{"u1"}, now)
			return []*domainContest.ContestTeamParticipation{p}, 1, nil
		},
	}
	teamRepo := &mockTeamRepo{
		findByIDsFn: func(_ []string) ([]*domainTeam.Team, error) {
			t := makeHandlerTeam("t1", "Alpha", "u1", now)
			return []*domainTeam.Team{t}, nil
		},
	}
	h := newHandlerWithListTeamRegistrations(
		&mockContestProvider{}, &mockGroupChecker{}, participRepo,
		teamRepo, &mockUserProvider{},
	)
	w := httptest.NewRecorder()
	r := listRegistrationsRequest("g1", "c1")
	wrapAuth(http.HandlerFunc(h.ListTeamRegistrations)).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body listTeamRegistrationsResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.Total != 1 {
		t.Errorf("Total = %d, want 1", body.Total)
	}
	if len(body.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(body.Items))
	}
	if body.Items[0].TeamID != "t1" {
		t.Errorf("Items[0].TeamID = %q, want t1", body.Items[0].TeamID)
	}
	if body.Items[0].TeamName != "Alpha" {
		t.Errorf("Items[0].TeamName = %q, want Alpha", body.Items[0].TeamName)
	}
	if len(body.Items[0].SelectedMembers) != 1 || body.Items[0].SelectedMembers[0].ID != "u1" {
		t.Errorf("unexpected SelectedMembers: %v", body.Items[0].SelectedMembers)
	}
}
