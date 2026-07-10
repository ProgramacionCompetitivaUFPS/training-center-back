package contest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appcontest "github.com/training-judge-center/backend/internal/application/contest"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

func newHandlerWithListMyContests(uc *appcontest.ListMyContestsUseCase) *Handler {
	return &Handler{listMyContests: uc}
}

func defaultListMyContestsUC() *appcontest.ListMyContestsUseCase {
	return appcontest.NewListMyContestsUseCase(
		&mockContestRepo{},
		&mockGroupProvider{},
		&mockContestParticipantProvider{},
	)
}

// mockListMyContestsRepo returns one fixed contest scoped to group "g1" from
// ListByGroupIDs, regardless of the groupIDs filter passed in.
type mockListMyContestsRepo struct{ mockContestRepo }

func (s *mockListMyContestsRepo) ListByGroupIDs(_ context.Context, _ []string, _ domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
	c := domainContest.RestoreContest(
		"c1",
		domainContest.RestoreContestName("Test Contest"),
		nil,
		time.Now().Add(24*time.Hour), time.Now().Add(29*time.Hour),
		domainContest.RestorePenalty(20),
		0, false, false, false,
		shared.RestoreGroupID("g1"),
		shared.RestoreUserID("u1"),
		domainContest.RestoreParticipationMode("INDIVIDUAL"), domainContest.RestoreTeamSize(2, 5),
		[]domainContest.ContestProblem{},
		time.Now(),
		nil,
	)
	return []*domainContest.Contest{c}, 1, nil
}

func TestListMyContests_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithListMyContests(defaultListMyContestsUC())
	r := httptest.NewRequest(http.MethodGet, "/contests", nil)
	w := httptest.NewRecorder()

	http.HandlerFunc(h.ListMyContests).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListMyContests_HappyPath_Returns200(t *testing.T) {
	h := newHandlerWithListMyContests(defaultListMyContestsUC())
	r := authedRequest(http.MethodGet, "/contests", nil)
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ListMyContests)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp listMyContestsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.Items == nil {
		t.Error("expected items array (possibly empty), got nil")
	}
}

func TestListMyContests_PaginationDefaults(t *testing.T) {
	h := newHandlerWithListMyContests(defaultListMyContestsUC())
	r := authedRequest(http.MethodGet, "/contests", nil)
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ListMyContests)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp listMyContestsResponse
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

func TestListMyContests_GroupFieldPresent(t *testing.T) {
	uc := appcontest.NewListMyContestsUseCase(&mockListMyContestsRepo{}, &mockGroupProvider{}, &mockContestParticipantProvider{})
	h := newHandlerWithListMyContests(uc)

	r := authedRequest(http.MethodGet, "/contests", nil)
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ListMyContests)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp listMyContestsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
	if resp.Items[0].Group.ID != "g1" || resp.Items[0].Group.Name != "Test Group" {
		t.Errorf("expected group {g1, Test Group}, got %+v", resp.Items[0].Group)
	}
}
