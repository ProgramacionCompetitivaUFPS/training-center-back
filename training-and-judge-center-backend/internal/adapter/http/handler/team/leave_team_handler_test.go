package team

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appTeam "github.com/training-judge-center/backend/internal/application/team"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newHandlerWithLeave(memberRepo domainTeam.MemberRepository) *Handler {
	return &Handler{leaveTeam: appTeam.NewLeaveTeamUseCase(memberRepo, &mockContestCheckerHandler{}, &mockScheduledParticipationCleanerHandler{}, &mockTxManager{})}
}

func TestLeaveTeam_UnauthenticatedReturns401(t *testing.T) {
	h := newHandlerWithLeave(&mockMemberRepo{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "/teams/t1/members/me", nil)
	wrapAuth(http.HandlerFunc(h.LeaveTeam)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestLeaveTeam_NotMemberReturns404(t *testing.T) {
	h := newHandlerWithLeave(&mockMemberRepo{})
	w := httptest.NewRecorder()
	r := authedGetRequest("/teams/t1/members/me")
	r.Method = "DELETE"
	wrapAuth(http.HandlerFunc(h.LeaveTeam)).ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestLeaveTeam_ActiveContestReturns409(t *testing.T) {
	memberRepo := &mockMemberRepo{
		findByTeamAndUserFn: func(_ string, userID domainShared.UserID) (*domainTeam.TeamMember, error) {
			return makeHandlerMember("m1", "t1", userID.Value(), time.Now()), nil
		},
	}
	checker := &mockContestCheckerHandler{
		inActiveFn: func(_, _ string) (bool, error) { return true, nil },
	}
	h := &Handler{leaveTeam: appTeam.NewLeaveTeamUseCase(memberRepo, checker, &mockScheduledParticipationCleanerHandler{}, &mockTxManager{})}
	w := httptest.NewRecorder()
	r := authedGetRequest("/teams/t1/members/me")
	r.Method = "DELETE"
	wrapAuth(http.HandlerFunc(h.LeaveTeam)).ServeHTTP(w, r)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.Code != domainTeam.ErrCodeCannotLeaveDuringActiveContest {
		t.Errorf("expected %s, got %s", domainTeam.ErrCodeCannotLeaveDuringActiveContest, body.Code)
	}
}

func TestLeaveTeam_SuccessReturns204(t *testing.T) {
	memberRepo := &mockMemberRepo{
		findByTeamAndUserFn: func(_ string, userID domainShared.UserID) (*domainTeam.TeamMember, error) {
			return makeHandlerMember("m1", "t1", userID.Value(), time.Now()), nil
		},
	}
	h := newHandlerWithLeave(memberRepo)
	w := httptest.NewRecorder()
	r := authedGetRequest("/teams/t1/members/me")
	r.Method = "DELETE"
	wrapAuth(http.HandlerFunc(h.LeaveTeam)).ServeHTTP(w, r)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d\nbody: %s", w.Code, w.Body.String())
	}
}
