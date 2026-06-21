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

func newHandlerWithGet(uc *appcontest.GetContestUseCase) *Handler {
	return &Handler{getContest: uc}
}

func defaultGetUC() *appcontest.GetContestUseCase {
	return appcontest.NewGetContestUseCase(
		&mockGetRepo{},
		&mockGroupProvider{},
		&mockMemberProvider{isLead: true, isMember: true},
		&mockProblemProvider{},
		&mockOwnerProvider{},
		&mockContestParticipantProvider{},
	)
}

// mockGetRepo returns a valid contest from FindByID.
type mockGetRepo struct{}

func (s *mockGetRepo) Create(_ context.Context, _ *domainContest.Contest) error { return nil }
func (s *mockGetRepo) Update(_ context.Context, _ *domainContest.Contest) error { return nil }
func (s *mockGetRepo) FindByID(_ context.Context, _ string) (*domainContest.Contest, error) {
	start := time.Now().Add(24 * time.Hour)
	end := time.Now().Add(29 * time.Hour)
	return domainContest.RestoreContest(
		"c1",
		domainContest.RestoreContestName("Test Contest"),
		nil,
		start, end,
		domainContest.RestorePenalty(20),
		0, false, false, false,
		shared.RestoreGroupID("g1"),
		shared.RestoreUserID("u1"),
		domainContest.RestoreParticipationMode("INDIVIDUAL"), domainContest.RestoreTeamSize(2, 5),
		[]domainContest.ContestProblem{},
		time.Now(),
		nil,
	), nil
}
func (s *mockGetRepo) Delete(_ context.Context, _ string) error { return nil }
func (s *mockGetRepo) List(_ context.Context, _ domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
	return nil, 0, nil
}

func TestGetContest_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithGet(defaultGetUC())
	r := httptest.NewRequest(http.MethodGet, "/groups/g1/contests/c1", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.Get).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetContest_HappyPath_Returns200(t *testing.T) {
	h := newHandlerWithGet(defaultGetUC())
	r := authedRequest(http.MethodGet, "/groups/g1/contests/c1", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Get)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp getContestResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.Name != "Test Contest" {
		t.Errorf("expected name 'Test Contest', got %q", resp.Name)
	}
	if resp.Status != "SCHEDULED" {
		t.Errorf("expected SCHEDULED, got %q", resp.Status)
	}
}

func TestGetContest_NotFound_Returns404(t *testing.T) {
	uc := appcontest.NewGetContestUseCase(
		&mockContestRepo{},
		&mockGroupProvider{},
		&mockMemberProvider{isLead: true, isMember: true},
		&mockProblemProvider{},
		&mockOwnerProvider{},
		&mockContestParticipantProvider{},
	)
	h := newHandlerWithGet(uc)
	r := authedRequest(http.MethodGet, "/groups/g1/contests/missing", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Get)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
