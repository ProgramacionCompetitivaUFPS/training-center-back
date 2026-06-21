package team

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	appTeam "github.com/training-judge-center/backend/internal/application/team"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newHandlerWithUpdateTeamRegistration(
	memberRepo domainTeam.MemberRepository,
	teamRepo domainTeam.Repository,
	contestProvider appTeam.ContestProvider,
	participRepo domainContest.TeamParticipationRepository,
	indivChecker appTeam.IndividualRegistrationChecker,
	groupChecker appTeam.GroupMemberChecker,
	userProvider appTeam.UserProvider,
	tx appshared.TransactionManager,
) *Handler {
	return &Handler{
		updateTeamRegistration: appTeam.NewUpdateTeamRegistrationUseCase(
			memberRepo, teamRepo, contestProvider, participRepo, indivChecker, groupChecker, userProvider, tx,
		),
	}
}

func updateRequest(contestID, teamID, body string) *http.Request {
	r := httptest.NewRequest("PUT", "/groups/g1/contests/"+contestID+"/team-registrations/"+teamID, strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer tok")
	r.Header.Set("Content-Type", "application/json")
	r.SetPathValue("contestId", contestID)
	r.SetPathValue("teamId", teamID)
	return r
}

func TestUpdateTeamRegistration_UnauthenticatedReturns401(t *testing.T) {
	h := newHandlerWithUpdateTeamRegistration(
		&mockMemberRepo{}, &mockTeamRepo{}, &mockContestProvider{},
		&mockTeamParticipRepo{}, &mockIndivChecker{}, &mockGroupChecker{},
		&mockUserProvider{}, &mockTxManager{},
	)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PUT", "/groups/g1/contests/c1/team-registrations/t1", nil)
	r.SetPathValue("contestId", "c1")
	r.SetPathValue("teamId", "t1")
	wrapAuth(http.HandlerFunc(h.UpdateTeamRegistration)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUpdateTeamRegistration_InvalidJSONReturns400(t *testing.T) {
	h := newHandlerWithUpdateTeamRegistration(
		&mockMemberRepo{}, &mockTeamRepo{}, &mockContestProvider{},
		&mockTeamParticipRepo{}, &mockIndivChecker{}, &mockGroupChecker{},
		&mockUserProvider{}, &mockTxManager{},
	)
	w := httptest.NewRecorder()
	r := updateRequest("c1", "t1", `{invalid json}`)
	wrapAuth(http.HandlerFunc(h.UpdateTeamRegistration)).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateTeamRegistration_RequesterNotMemberReturns403(t *testing.T) {
	h := newHandlerWithUpdateTeamRegistration(
		&mockMemberRepo{}, &mockTeamRepo{}, &mockContestProvider{},
		&mockTeamParticipRepo{}, &mockIndivChecker{}, &mockGroupChecker{},
		&mockUserProvider{}, &mockTxManager{},
	)
	w := httptest.NewRecorder()
	r := updateRequest("c1", "t1", `{"selectedMembers":["u1"]}`)
	wrapAuth(http.HandlerFunc(h.UpdateTeamRegistration)).ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.Code != domainTeam.ErrCodeNotTeamMember {
		t.Errorf("expected %s, got %s", domainTeam.ErrCodeNotTeamMember, body.Code)
	}
}

func TestUpdateTeamRegistration_ContestAlreadyStartedReturns409(t *testing.T) {
	now := time.Now()
	memberRepo := &mockMemberRepo{
		findByTeamAndUserFn: func(_ string, userID domainShared.UserID) (*domainTeam.TeamMember, error) {
			return makeHandlerMember("m1", "t1", userID.Value(), now), nil
		},
	}
	contestProvider := &mockContestProvider{
		getInfoFn: func(id string) (*appTeam.ContestInfo, error) {
			return &appTeam.ContestInfo{
				ID: id, GroupID: "g1",
				ParticipationMode: "TEAM",
				TeamSizeMin: 1, TeamSizeMax: 3, Status: "ACTIVE",
			}, nil
		},
	}
	h := newHandlerWithUpdateTeamRegistration(
		memberRepo, &mockTeamRepo{}, contestProvider,
		&mockTeamParticipRepo{}, &mockIndivChecker{}, &mockGroupChecker{},
		&mockUserProvider{}, &mockTxManager{},
	)
	w := httptest.NewRecorder()
	r := updateRequest("c1", "t1", `{"selectedMembers":["u1"]}`)
	wrapAuth(http.HandlerFunc(h.UpdateTeamRegistration)).ServeHTTP(w, r)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.Code != domainContest.ErrCodeContestAlreadyStarted {
		t.Errorf("expected %s, got %s", domainContest.ErrCodeContestAlreadyStarted, body.Code)
	}
}

func TestUpdateTeamRegistration_SuccessReturns200(t *testing.T) {
	now := time.Now()
	memberRepo := &mockMemberRepo{
		findByTeamAndUserFn: func(_ string, userID domainShared.UserID) (*domainTeam.TeamMember, error) {
			return makeHandlerMember("m1", "t1", userID.Value(), now), nil
		},
		findByTeamFn: func(_ string) ([]*domainTeam.TeamMember, error) {
			return []*domainTeam.TeamMember{
				makeHandlerMember("m1", "t1", "u1", now),
			}, nil
		},
	}
	teamRepo := &mockTeamRepo{
		findByIDFn: func(_ string) (*domainTeam.Team, error) {
			return makeHandlerTeam("t1", "Alpha", "u1", now), nil
		},
	}
	h := newHandlerWithUpdateTeamRegistration(
		memberRepo, teamRepo, &mockContestProvider{},
		&mockTeamParticipRepo{}, &mockIndivChecker{}, &mockGroupChecker{},
		&mockUserProvider{}, &mockTxManager{},
	)
	w := httptest.NewRecorder()
	r := updateRequest("c1", "t1", `{"selectedMembers":["u1"]}`)
	wrapAuth(http.HandlerFunc(h.UpdateTeamRegistration)).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body updateTeamRegistrationResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.TeamID != "t1" {
		t.Errorf("TeamID = %q, want t1", body.TeamID)
	}
	if body.TeamName != "Alpha" {
		t.Errorf("TeamName = %q, want Alpha", body.TeamName)
	}
	if len(body.SelectedMembers) != 1 || body.SelectedMembers[0].ID != "u1" {
		t.Errorf("unexpected SelectedMembers: %v", body.SelectedMembers)
	}
}
