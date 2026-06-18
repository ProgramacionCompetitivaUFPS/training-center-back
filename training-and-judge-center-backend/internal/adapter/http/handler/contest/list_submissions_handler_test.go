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

// ── repo returning a specific contest ────────────────────────────────────────

type mockSubmissionsContestRepo struct {
	contest *domainContest.Contest
}

func (m *mockSubmissionsContestRepo) Create(_ context.Context, _ *domainContest.Contest) error { return nil }
func (m *mockSubmissionsContestRepo) Update(_ context.Context, _ *domainContest.Contest) error { return nil }
func (m *mockSubmissionsContestRepo) FindByID(_ context.Context, _ string) (*domainContest.Contest, error) {
	return m.contest, nil
}
func (m *mockSubmissionsContestRepo) Delete(_ context.Context, _ string) error { return nil }
func (m *mockSubmissionsContestRepo) List(_ context.Context, _ domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
	return nil, 0, nil
}

// ── participant provider always registered ────────────────────────────────────

type mockParticipantRegistered struct{}

func (m *mockParticipantRegistered) IsRegistered(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}
func (m *mockParticipantRegistered) CountParticipants(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (m *mockParticipantRegistered) CountParticipantsBulk(_ context.Context, contestIDs []string) (map[string]int, error) {
	result := make(map[string]int, len(contestIDs))
	for _, id := range contestIDs {
		result[id] = 0
	}
	return result, nil
}
func (m *mockParticipantRegistered) IsRegisteredBulk(_ context.Context, contestIDs []string, _ string) (map[string]bool, error) {
	result := make(map[string]bool, len(contestIDs))
	for _, id := range contestIDs {
		result[id] = true
	}
	return result, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func activeContestForSubmissions() *domainContest.Contest {
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
		domainContest.RestoreParticipationMode("INDIVIDUAL"), domainContest.RestoreTeamSize(2, 5),
		[]domainContest.ContestProblem{},
		time.Now(), nil,
	)
}

func newHandlerWithListSubmissions(uc *appcontest.ListContestSubmissionsUseCase) *Handler {
	return &Handler{listContestSubmissions: uc}
}

func defaultListSubmissionsUC(subs []appcontest.RichSubmissionData) *appcontest.ListContestSubmissionsUseCase {
	return appcontest.NewListContestSubmissionsUseCase(
		&mockSubmissionsContestRepo{contest: activeContestForSubmissions()},
		&mockGroupProvider{},
		&mockMemberProvider{isLead: true, isMember: true},
		&mockParticipantRegistered{},
		&mockContestSubmissionsProvider{subs: subs},
	)
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestListContestSubmissions_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithListSubmissions(defaultListSubmissionsUC(nil))
	r := httptest.NewRequest(http.MethodGet, "/groups/g1/contests/c1/submissions", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.ListContestSubmissions).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListContestSubmissions_HappyPath_Returns200(t *testing.T) {
	subs := []appcontest.RichSubmissionData{
		{
			ID:           "sub-1",
			ProblemSlug:  "sum",
			ProblemTitle: "Sum of Two Numbers",
			ProblemOrder: 1,
			UserID:       "u1",
			Nickname:     "john_doe",
			Language:     "cpp20",
			Status:       "ACCEPTED",
			SubmittedAt:  time.Now().Add(-30 * time.Minute),
		},
	}
	h := newHandlerWithListSubmissions(defaultListSubmissionsUC(subs))
	r := authedRequest(http.MethodGet, "/groups/g1/contests/c1/submissions", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ListContestSubmissions)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp listSubmissionsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Submissions) != 1 {
		t.Errorf("submissions=%d, want 1", len(resp.Submissions))
	}
	if resp.Submissions[0].Status != "ACCEPTED" {
		t.Errorf("status=%q, want ACCEPTED", resp.Submissions[0].Status)
	}
	if resp.Submissions[0].Problem.Slug != "sum" {
		t.Errorf("slug=%q, want sum", resp.Submissions[0].Problem.Slug)
	}
	if resp.Contest.Status != "ACTIVE" {
		t.Errorf("contest.status=%q, want ACTIVE", resp.Contest.Status)
	}
	if resp.Pagination.Total != 1 {
		t.Errorf("total=%d, want 1", resp.Pagination.Total)
	}
}

func TestListContestSubmissions_Pagination_Returns200(t *testing.T) {
	subs := make([]appcontest.RichSubmissionData, 5)
	for i := range subs {
		subs[i] = appcontest.RichSubmissionData{
			ID:          "sub-" + string(rune('1'+i)),
			ProblemSlug: "sum",
			UserID:      "u1",
			Nickname:    "john_doe",
			Language:    "cpp20",
			Status:      "ACCEPTED",
			SubmittedAt: time.Now().Add(-time.Duration(i+1) * 10 * time.Minute),
		}
	}
	h := newHandlerWithListSubmissions(defaultListSubmissionsUC(subs))
	r := authedRequest(http.MethodGet, "/groups/g1/contests/c1/submissions?page=2&limit=2", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ListContestSubmissions)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp listSubmissionsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Pagination.Total != 5 {
		t.Errorf("total=%d, want 5", resp.Pagination.Total)
	}
	if resp.Pagination.TotalPages != 3 {
		t.Errorf("totalPages=%d, want 3", resp.Pagination.TotalPages)
	}
	if len(resp.Submissions) != 2 {
		t.Errorf("page entries=%d, want 2", len(resp.Submissions))
	}
}
