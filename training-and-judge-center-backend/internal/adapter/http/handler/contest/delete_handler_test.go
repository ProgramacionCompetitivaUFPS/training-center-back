package contest

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appcontest "github.com/training-judge-center/backend/internal/application/contest"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

func contestToDelete() *domainContest.Contest {
	return domainContest.RestoreContest(
		"c1",
		domainContest.RestoreContestName("Contest To Delete"),
		nil,
		time.Now().Add(2*time.Hour),
		time.Now().Add(4*time.Hour),
		domainContest.RestorePenalty(20),
		0, false, false, false,
		shared.RestoreGroupID("g1"),
		shared.RestoreUserID("u1"),
		domainContest.RestoreParticipationMode("INDIVIDUAL"), domainContest.RestoreTeamSize(2, 5),
		[]domainContest.ContestProblem{},
		time.Now(), nil,
	)
}

func newHandlerWithDelete(uc *appcontest.DeleteContestUseCase) *Handler {
	return &Handler{deleteContest: uc}
}

func defaultDeleteUC() *appcontest.DeleteContestUseCase {
	return appcontest.NewDeleteContestUseCase(
		&mockRepoReturning{contest: contestToDelete()},
		&mockGroupProvider{},
		&mockMemberProvider{isLead: true, isMember: true},
		&mockStandingsCache{},
	)
}

func TestDeleteContest_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithDelete(defaultDeleteUC())
	r := httptest.NewRequest(http.MethodDelete, "/groups/g1/contests/c1", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.Delete).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestDeleteContest_Lead_Returns204(t *testing.T) {
	h := newHandlerWithDelete(defaultDeleteUC())
	r := authedRequest(http.MethodDelete, "/groups/g1/contests/c1", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Delete)).ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteContest_ContestNotFound_Returns404(t *testing.T) {
	uc := appcontest.NewDeleteContestUseCase(
		&mockRepoReturning{contest: nil},
		&mockGroupProvider{},
		&mockMemberProvider{isLead: true, isMember: true},
		&mockStandingsCache{},
	)
	h := newHandlerWithDelete(uc)
	r := authedRequest(http.MethodDelete, "/groups/g1/contests/c1", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Delete)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
