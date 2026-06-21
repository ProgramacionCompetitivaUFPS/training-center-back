package team

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appTeam "github.com/training-judge-center/backend/internal/application/team"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newHandlerWithUnregisterTeamFromContest(
	memberRepo domainTeam.MemberRepository,
	contestProvider appTeam.ContestProvider,
	participRepo domainContest.TeamParticipationRepository,
) *Handler {
	return &Handler{
		unregisterTeamFromContest: appTeam.NewUnregisterTeamFromContestUseCase(
			memberRepo, contestProvider, participRepo,
		),
	}
}

func deleteRegistrationRequest(contestID, teamID string) *http.Request {
	r := httptest.NewRequest("DELETE", "/groups/g1/contests/"+contestID+"/team-registrations/"+teamID, nil)
	r.Header.Set("Authorization", "Bearer tok")
	r.SetPathValue("contestId", contestID)
	r.SetPathValue("teamId", teamID)
	return r
}

func TestUnregisterTeamFromContest_UnauthenticatedReturns401(t *testing.T) {
	h := newHandlerWithUnregisterTeamFromContest(&mockMemberRepo{}, &mockContestProvider{}, &mockTeamParticipRepo{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "/groups/g1/contests/c1/team-registrations/t1", nil)
	r.SetPathValue("contestId", "c1")
	r.SetPathValue("teamId", "t1")
	wrapAuth(http.HandlerFunc(h.UnregisterTeamFromContest)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUnregisterTeamFromContest_RequesterNotMemberReturns403(t *testing.T) {
	h := newHandlerWithUnregisterTeamFromContest(&mockMemberRepo{}, &mockContestProvider{}, &mockTeamParticipRepo{})
	w := httptest.NewRecorder()
	r := deleteRegistrationRequest("c1", "t1")
	wrapAuth(http.HandlerFunc(h.UnregisterTeamFromContest)).ServeHTTP(w, r)
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

func TestUnregisterTeamFromContest_ContestAlreadyStartedReturns409(t *testing.T) {
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
	h := newHandlerWithUnregisterTeamFromContest(memberRepo, contestProvider, &mockTeamParticipRepo{})
	w := httptest.NewRecorder()
	r := deleteRegistrationRequest("c1", "t1")
	wrapAuth(http.HandlerFunc(h.UnregisterTeamFromContest)).ServeHTTP(w, r)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.Code != domainContest.ErrCodeCannotUnregisterAfterStart {
		t.Errorf("expected %s, got %s", domainContest.ErrCodeCannotUnregisterAfterStart, body.Code)
	}
}

func TestUnregisterTeamFromContest_SuccessReturns204(t *testing.T) {
	now := time.Now()
	memberRepo := &mockMemberRepo{
		findByTeamAndUserFn: func(_ string, userID domainShared.UserID) (*domainTeam.TeamMember, error) {
			return makeHandlerMember("m1", "t1", userID.Value(), now), nil
		},
	}
	h := newHandlerWithUnregisterTeamFromContest(memberRepo, &mockContestProvider{}, &mockTeamParticipRepo{})
	w := httptest.NewRecorder()
	r := deleteRegistrationRequest("c1", "t1")
	wrapAuth(http.HandlerFunc(h.UnregisterTeamFromContest)).ServeHTTP(w, r)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d\nbody: %s", w.Code, w.Body.String())
	}
}
