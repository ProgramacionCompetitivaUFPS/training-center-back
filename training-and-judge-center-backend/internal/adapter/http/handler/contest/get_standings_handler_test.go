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

// ── handler-scoped mocks ──────────────────────────────────────────────────────

type mockStandingsContestRepo struct {
	contest *domainContest.Contest
}

func (m *mockStandingsContestRepo) Create(_ context.Context, _ *domainContest.Contest) error { return nil }
func (m *mockStandingsContestRepo) Update(_ context.Context, _ *domainContest.Contest) error { return nil }
func (m *mockStandingsContestRepo) FindByID(_ context.Context, _ string) (*domainContest.Contest, error) {
	return m.contest, nil
}
func (m *mockStandingsContestRepo) Delete(_ context.Context, _ string) error { return nil }
func (m *mockStandingsContestRepo) List(_ context.Context, _ domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
	return nil, 0, nil
}

type mockHandlerStandingsCache struct {
	data *appcontest.CachedStandings
}

func (m *mockHandlerStandingsCache) Get(_ context.Context, _ string) (*appcontest.CachedStandings, error) {
	return m.data, nil
}
func (m *mockHandlerStandingsCache) Set(_ context.Context, _ string, _ *appcontest.CachedStandings) error {
	return nil
}
func (m *mockHandlerStandingsCache) AcquireRefreshLock(_ context.Context, _ string, _ time.Duration) (bool, error) {
	return true, nil
}
func (m *mockHandlerStandingsCache) ReleaseRefreshLock(_ context.Context, _ string) error {
	return nil
}

type mockHandlerSubProvider struct{}

func (m *mockHandlerSubProvider) ListByContest(_ context.Context, _ string) ([]appcontest.ContestSubmissionData, error) {
	return nil, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newHandlerWithGetStandings(uc *appcontest.GetStandingsUseCase) *Handler {
	return &Handler{getStandings: uc}
}

func activeContestForStandings() *domainContest.Contest {
	return domainContest.RestoreContest(
		"c1",
		domainContest.RestoreContestName("Test Contest"),
		nil,
		time.Now().Add(-2*time.Hour),
		time.Now().Add(2*time.Hour),
		domainContest.RestorePenalty(20),
		0, false, false,
		shared.RestoreGroupID("g1"),
		shared.RestoreUserID("u1"),
		[]domainContest.ContestProblem{},
		time.Now(),
		nil,
	)
}

func defaultGetStandingsUC() *appcontest.GetStandingsUseCase {
	accepted := time.Now().Add(-60 * time.Minute)
	cached := &appcontest.CachedStandings{
		Participants: []domainContest.ParticipantStanding{
			{ContestantID: "u1", Problems: map[string]domainContest.ProblemAttempt{
				"A": {Attempts: 0, AcceptedAt: &accepted},
			}},
		},
		LastUpdated: time.Now(),
	}
	return appcontest.NewGetStandingsUseCase(
		&mockStandingsContestRepo{contest: activeContestForStandings()},
		&mockRegistrationRepository{},
		&mockHandlerSubProvider{},
		&mockGroupProvider{},
		&mockMemberProvider{isLead: true, isMember: true},
		&mockHandlerStandingsCache{data: cached},
		30*time.Second,
	)
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestGetStandings_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithGetStandings(defaultGetStandingsUC())
	r := httptest.NewRequest(http.MethodGet, "/groups/g1/contests/c1/standings", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.GetStandings).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetStandings_HappyPath_Returns200(t *testing.T) {
	h := newHandlerWithGetStandings(defaultGetStandingsUC())
	r := authedRequest(http.MethodGet, "/groups/g1/contests/c1/standings", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GetStandings)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp getStandingsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Entries) != 1 {
		t.Errorf("entries=%d, want 1", len(resp.Entries))
	}
	if resp.Entries[0].ProblemsSolved != 1 {
		t.Errorf("problemsSolved=%d, want 1", resp.Entries[0].ProblemsSolved)
	}
	if resp.Pagination.Total != 1 {
		t.Errorf("total=%d, want 1", resp.Pagination.Total)
	}
	if resp.Meta.ContestStatus != "ACTIVE" {
		t.Errorf("contestStatus=%q, want ACTIVE", resp.Meta.ContestStatus)
	}
}

func TestGetStandings_Pagination_Returns200(t *testing.T) {
	cached := &appcontest.CachedStandings{
		Participants: []domainContest.ParticipantStanding{
			{ContestantID: "u1", Problems: map[string]domainContest.ProblemAttempt{}},
			{ContestantID: "u2", Problems: map[string]domainContest.ProblemAttempt{}},
			{ContestantID: "u3", Problems: map[string]domainContest.ProblemAttempt{}},
		},
		LastUpdated: time.Now(),
	}
	uc := appcontest.NewGetStandingsUseCase(
		&mockStandingsContestRepo{contest: activeContestForStandings()},
		&mockRegistrationRepository{},
		&mockHandlerSubProvider{},
		&mockGroupProvider{},
		&mockMemberProvider{isLead: true, isMember: true},
		&mockHandlerStandingsCache{data: cached},
		30*time.Second,
	)
	h := newHandlerWithGetStandings(uc)

	r := authedRequest(http.MethodGet, "/groups/g1/contests/c1/standings?page=2&limit=1", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GetStandings)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp getStandingsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Pagination.Total != 3 {
		t.Errorf("total=%d, want 3", resp.Pagination.Total)
	}
	if resp.Pagination.TotalPages != 3 {
		t.Errorf("totalPages=%d, want 3", resp.Pagination.TotalPages)
	}
	if len(resp.Entries) != 1 {
		t.Errorf("page entries=%d, want 1", len(resp.Entries))
	}
}
