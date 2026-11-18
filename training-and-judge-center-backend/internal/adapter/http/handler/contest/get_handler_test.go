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
func (s *mockGetRepo) ListByGroupIDs(_ context.Context, _ []string, _ domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
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
	if resp.ParticipationMode != "INDIVIDUAL" || resp.TeamSizeMin != 2 || resp.TeamSizeMax != 5 {
		t.Errorf("expected INDIVIDUAL with team size 2–5, got %q %d–%d",
			resp.ParticipationMode, resp.TeamSizeMin, resp.TeamSizeMax)
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

type coloredGetRepo struct{ mockGetRepo }

func (s *coloredGetRepo) FindByID(ctx context.Context, id string) (*domainContest.Contest, error) {
	c, err := s.mockGetRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	red := domainContest.RestoreProblemColor("#aabbcc")
	c.SetProblems([]domainContest.ContestProblem{
		domainContest.RestoreContestProblem("cp1", "p1", 1, &red),
		domainContest.RestoreContestProblem("cp2", "p2", 2, nil),
	}, time.Now())
	return c, nil
}

func TestGetContest_ProblemsIncludeColorOrNull(t *testing.T) {
	uc := appcontest.NewGetContestUseCase(
		&coloredGetRepo{},
		&mockGroupProvider{},
		&mockMemberProvider{isLead: true, isMember: true},
		&mockProblemProvider{},
		&mockOwnerProvider{},
		&mockContestParticipantProvider{},
	)
	r := authedRequest(http.MethodGet, "/groups/g1/contests/c1", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(newHandlerWithGet(uc).Get)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var raw struct {
		Problems []map[string]interface{} `json:"problems"`
	}
	if err := json.NewDecoder(w.Body).Decode(&raw); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if len(raw.Problems) != 2 {
		t.Fatalf("expected 2 problems, got %d", len(raw.Problems))
	}
	if got := raw.Problems[0]["color"]; got != "#aabbcc" {
		t.Errorf("problem 1 color = %v, want #aabbcc", got)
	}
	if got, present := raw.Problems[1]["color"]; !present || got != nil {
		t.Errorf("problem 2 color = %v (present=%v), want an explicit null", got, present)
	}
}
