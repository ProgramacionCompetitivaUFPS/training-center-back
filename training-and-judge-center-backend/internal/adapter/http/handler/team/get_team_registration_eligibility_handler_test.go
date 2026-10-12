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

func newHandlerWithGetTeamRegistrationEligibility(
	memberRepo domainTeam.MemberRepository,
	groupChecker appTeam.GroupMemberChecker,
) *Handler {
	return &Handler{
		getTeamRegistrationEligibility: appTeam.NewGetTeamRegistrationEligibilityUseCase(
			memberRepo, &mockContestProvider{}, unregisteredTeamParticipRepo(),
			&mockIndivChecker{}, groupChecker, &mockUserProvider{},
		),
	}
}

func unregisteredTeamParticipRepo() *mockTeamParticipRepo {
	return &mockTeamParticipRepo{
		findByContestAndTeamFn: func(_, _ string) (*domainContest.ContestTeamParticipation, error) {
			return nil, apperror.NewNotFound(domainContest.ErrCodeTeamRegistrationNotFound, "team registration not found")
		},
	}
}

func eligibilityRequest(groupID, contestID, teamID string) *http.Request {
	r := httptest.NewRequest("GET", "/groups/"+groupID+"/contests/"+contestID+"/team-registrations/"+teamID+"/eligibility", nil)
	r.Header.Set("Authorization", "Bearer tok")
	r.SetPathValue("groupId", groupID)
	r.SetPathValue("contestId", contestID)
	r.SetPathValue("teamId", teamID)
	return r
}

func rosterMemberRepo(userIDs ...string) *mockMemberRepo {
	now := time.Now()
	return &mockMemberRepo{
		findByTeamAndUserFn: func(teamID string, userID domainShared.UserID) (*domainTeam.TeamMember, error) {
			return makeHandlerMember("m-"+userID.Value(), teamID, userID.Value(), now), nil
		},
		findByTeamFn: func(teamID string) ([]*domainTeam.TeamMember, error) {
			members := make([]*domainTeam.TeamMember, len(userIDs))
			for i, id := range userIDs {
				members[i] = makeHandlerMember("m-"+id, teamID, id, now)
			}
			return members, nil
		},
	}
}

func TestGetTeamRegistrationEligibility_UnauthenticatedReturns401(t *testing.T) {
	h := newHandlerWithGetTeamRegistrationEligibility(rosterMemberRepo("u1"), &mockGroupChecker{})
	w := httptest.NewRecorder()
	r := eligibilityRequest("g1", "c1", "t1")
	r.Header.Del("Authorization")
	wrapAuth(http.HandlerFunc(h.GetTeamRegistrationEligibility)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetTeamRegistrationEligibility_NotTeamMemberReturns403(t *testing.T) {
	h := newHandlerWithGetTeamRegistrationEligibility(&mockMemberRepo{}, &mockGroupChecker{})
	w := httptest.NewRecorder()
	wrapAuth(http.HandlerFunc(h.GetTeamRegistrationEligibility)).ServeHTTP(w, eligibilityRequest("g1", "c1", "t1"))
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetTeamRegistrationEligibility_SuccessReturns200(t *testing.T) {
	groupChecker := &mockGroupChecker{
		fn: func(_ string, _ []string) (map[string]bool, error) {
			return map[string]bool{"u1": true, "u2": false}, nil
		},
	}
	h := newHandlerWithGetTeamRegistrationEligibility(rosterMemberRepo("u1", "u2"), groupChecker)
	w := httptest.NewRecorder()
	wrapAuth(http.HandlerFunc(h.GetTeamRegistrationEligibility)).ServeHTTP(w, eligibilityRequest("g1", "c1", "t1"))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body teamRegistrationEligibilityResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.TeamAlreadyRegistered {
		t.Errorf("TeamAlreadyRegistered = true, want false")
	}
	if len(body.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(body.Members))
	}
	if !body.Members[0].Eligible || body.Members[0].Reason != "" {
		t.Errorf("Members[0] = %+v, want eligible with no reason", body.Members[0])
	}
	if body.Members[1].Eligible || body.Members[1].Reason != appTeam.ErrCodeMemberNotInGroup {
		t.Errorf("Members[1] = %+v, want not eligible with %s", body.Members[1], appTeam.ErrCodeMemberNotInGroup)
	}
}
