package team

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newHandlerWithRemoveMember(memberRepo domainTeam.MemberRepository, checker appTeam.ContestParticipationChecker) *Handler {
	return &Handler{removeTeamMember: appTeam.NewRemoveTeamMemberUseCase(memberRepo, checker, &mockScheduledParticipationCleanerHandler{}, &mockTxManager{})}
}

func removeRequest(teamID, userID string) *http.Request {
	r := authedGetRequest("/teams/" + teamID + "/members/" + userID)
	r.Method = http.MethodDelete
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("teamId", teamID)
	chiCtx.URLParams.Add("userId", userID)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chiCtx))
}

// u1 (the authenticated user) and "u2" are both on the team.
func u1AndU2OnTheTeam() *mockMemberRepo {
	return &mockMemberRepo{
		findByTeamAndUserFn: func(teamID string, userID domainShared.UserID) (*domainTeam.TeamMember, error) {
			if userID.Value() == "u1" || userID.Value() == "u2" {
				return makeHandlerMember("m-"+userID.Value(), teamID, userID.Value(), time.Now()), nil
			}
			return nil, apperror.NewNotFound(domainTeam.ErrCodeNotTeamMember, "not a member")
		},
	}
}

func TestRemoveTeamMember_UnauthenticatedReturns401(t *testing.T) {
	h := newHandlerWithRemoveMember(&mockMemberRepo{}, &mockContestCheckerHandler{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/teams/t1/members/u2", nil)
	wrapAuth(http.HandlerFunc(h.RemoveTeamMember)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRemoveTeamMember_RemovesTheOtherMemberAndReturns204(t *testing.T) {
	memberRepo := u1AndU2OnTheTeam()
	var deleted string
	memberRepo.deleteByTeamAndUserFn = func(_ string, userID domainShared.UserID) error {
		deleted = userID.Value()
		return nil
	}
	h := newHandlerWithRemoveMember(memberRepo, &mockContestCheckerHandler{})
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.RemoveTeamMember)).ServeHTTP(w, removeRequest("t1", "u2"))

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d\nbody: %s", w.Code, w.Body.String())
	}
	if deleted != "u2" {
		t.Errorf("expected u2 to be removed, got %q", deleted)
	}
}

func TestRemoveTeamMember_RemovingYourselfReturns400(t *testing.T) {
	h := newHandlerWithRemoveMember(u1AndU2OnTheTeam(), &mockContestCheckerHandler{})
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.RemoveTeamMember)).ServeHTTP(w, removeRequest("t1", "u1"))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRemoveTeamMember_UnknownTargetReturns404(t *testing.T) {
	h := newHandlerWithRemoveMember(u1AndU2OnTheTeam(), &mockContestCheckerHandler{})
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.RemoveTeamMember)).ServeHTTP(w, removeRequest("t1", "u9"))

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestRemoveTeamMember_TargetInActiveContestReturns409(t *testing.T) {
	checker := &mockContestCheckerHandler{inActiveFn: func(userID, _ string) (bool, error) { return userID == "u2", nil }}
	h := newHandlerWithRemoveMember(u1AndU2OnTheTeam(), checker)
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.RemoveTeamMember)).ServeHTTP(w, removeRequest("t1", "u2"))

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.Code != domainTeam.ErrCodeCannotRemoveDuringActiveContest {
		t.Errorf("expected %s, got %s", domainTeam.ErrCodeCannotRemoveDuringActiveContest, body.Code)
	}
}
