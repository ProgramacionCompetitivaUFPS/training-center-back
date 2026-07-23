package contest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	appcontest "github.com/training-judge-center/backend/internal/application/contest"
)

func newHandlerWithUnregister(uc *appcontest.UnregisterFromContestUseCase) *Handler {
	return &Handler{unregisterFromContest: uc}
}

func defaultUnregisterUC() *appcontest.UnregisterFromContestUseCase {
	return appcontest.NewUnregisterFromContestUseCase(
		&mockRepoReturning{contest: scheduledContest()},
		&mockRegistrationRepository{},
		&mockMemberProvider{isLead: false, isMember: true},
	)
}

func TestUnregister_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithUnregister(defaultUnregisterUC())
	r := httptest.NewRequest(http.MethodDelete, "/groups/g1/contests/c1/register", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.Unregister).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUnregister_HappyPath_Returns204(t *testing.T) {
	h := newHandlerWithUnregister(defaultUnregisterUC())
	r := authedRequest(http.MethodDelete, "/groups/g1/contests/c1/register", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Unregister)).ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUnregister_ContestNotFound_Returns404(t *testing.T) {
	uc := appcontest.NewUnregisterFromContestUseCase(
		&mockRepoReturning{contest: nil},
		&mockRegistrationRepository{},
		&mockMemberProvider{isMember: true},
	)
	h := newHandlerWithUnregister(uc)
	r := authedRequest(http.MethodDelete, "/groups/g1/contests/missing/register", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Unregister)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUnregister_NonMember_Returns403(t *testing.T) {
	uc := appcontest.NewUnregisterFromContestUseCase(
		&mockRepoReturning{contest: scheduledContest()},
		&mockRegistrationRepository{},
		&mockMemberProvider{isLead: false, isMember: false},
	)
	h := newHandlerWithUnregister(uc)
	r := authedRequest(http.MethodDelete, "/groups/g1/contests/c1/register", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Unregister)).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUnregister_ContestAlreadyStarted_Returns400(t *testing.T) {
	uc := appcontest.NewUnregisterFromContestUseCase(
		&mockRepoReturning{contest: finishedContest()},
		&mockRegistrationRepository{},
		&mockMemberProvider{isMember: true},
	)
	h := newHandlerWithUnregister(uc)
	r := authedRequest(http.MethodDelete, "/groups/g1/contests/c1/register", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Unregister)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
