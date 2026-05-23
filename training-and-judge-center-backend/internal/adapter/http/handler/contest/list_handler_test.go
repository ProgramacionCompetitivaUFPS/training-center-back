package contest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appcontest "github.com/training-judge-center/backend/internal/application/contest"
)

func newHandlerWithList(uc *appcontest.ListContestsUseCase) *Handler {
	return &Handler{listContests: uc}
}

func defaultListUC() *appcontest.ListContestsUseCase {
	return appcontest.NewListContestsUseCase(
		&mockContestRepo{},
		&mockGroupProvider{},
		&mockMemberProvider{isLead: false, isMember: true},
		&mockContestParticipantProvider{},
	)
}

func TestListContests_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithList(defaultListUC())
	r := httptest.NewRequest(http.MethodGet, "/groups/g1/contests", nil)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.List).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListContests_HappyPath_Returns200(t *testing.T) {
	h := newHandlerWithList(defaultListUC())
	r := authedRequest(http.MethodGet, "/groups/g1/contests", nil)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.List)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp listContestsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.Items == nil {
		t.Error("expected items array (possibly empty), got nil")
	}
}

func TestListContests_MissingGroupId_Returns400(t *testing.T) {
	h := newHandlerWithList(defaultListUC())
	r := authedRequest(http.MethodGet, "/groups//contests", nil)
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.List)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestListContests_PaginationDefaults(t *testing.T) {
	h := newHandlerWithList(defaultListUC())
	r := authedRequest(http.MethodGet, "/groups/g1/contests", nil)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.List)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp listContestsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.Pagination.Page != 1 {
		t.Errorf("expected page %d, got %d", 1, resp.Pagination.Page)
	}
	if resp.Pagination.Limit != 20 {
		t.Errorf("expected limit %d, got %d", 20, resp.Pagination.Limit)
	}
}
