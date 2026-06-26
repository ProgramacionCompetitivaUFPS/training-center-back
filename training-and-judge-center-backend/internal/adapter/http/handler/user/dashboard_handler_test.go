package user

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
)

func newHandlerWithGetDashboard(uc *appuser.GetDashboardUseCase) *Handler {
	return &Handler{getDashboard: uc}
}

func TestGetDashboard_Unauthenticated_Returns401(t *testing.T) {
	uc := appuser.NewGetDashboardUseCase(nil, nil, nil, nil)
	h := newHandlerWithGetDashboard(uc)

	req := httptest.NewRequest(http.MethodGet, "/users/me/dashboard", nil)
	rr := httptest.NewRecorder()

	h.GetDashboard(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestGetDashboard_Authenticated_EmptyData_Returns200(t *testing.T) {
	uc := appuser.NewGetDashboardUseCase(
		&mockDashboardSubmissionProvider{},
		&mockDashboardContestProvider{},
		&mockDashboardMaterialProvider{},
		&mockDashboardRankingProvider{},
	)
	h := newHandlerWithGetDashboard(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.GetDashboard),
		&domainuser.TokenClaims{UserID: "user-1", Role: shared.RoleContestant},
	)

	req := httptest.NewRequest(http.MethodGet, "/users/me/dashboard", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}

	var resp dashboardResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.RecentSubmissions == nil {
		t.Error("recentSubmissions should be an empty array, not null")
	}
	if resp.UpcomingContests == nil {
		t.Error("upcomingContests should be an empty array, not null")
	}
	if resp.ActiveContests == nil {
		t.Error("activeContests should be an empty array, not null")
	}
	if resp.RecentMaterials == nil {
		t.Error("recentMaterials should be an empty array, not null")
	}
	if resp.RecentContestResults == nil {
		t.Error("recentContestResults should be an empty array, not null")
	}
	if resp.ProblemsSolved != 0 {
		t.Errorf("problemsSolved: want 0, got %d", resp.ProblemsSolved)
	}
	if resp.Ranking.Position != nil {
		t.Errorf("ranking.position: want null, got %v", resp.Ranking.Position)
	}
}

func TestGetDashboard_Authenticated_PopulatedData_Returns200(t *testing.T) {
	const userID = "user-42"
	execTime := 120
	memKb := 4096
	pos := 3

	fixedTime := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)

	subProvider := &mockDashboardSubmissionProvider{
		getRecentSubmissionsFn: func(_ context.Context, _ string, _ int) ([]appuser.DashboardSubmission, error) {
			return []appuser.DashboardSubmission{
				{
					ID:            "sub-1",
					ProblemSlug:   "two-sum",
					ProblemTitle:  "Two Sum",
					Verdict:       "ACCEPTED",
					Language:      "go",
					SubmittedAt:   fixedTime,
					ExecutionTime: &execTime,
					MemoryKb:      &memKb,
				},
			}, nil
		},
		getSubmissionDatesFn: func(_ context.Context, _ string) ([]time.Time, error) {
			return []time.Time{fixedTime}, nil
		},
	}
	contestProvider := &mockDashboardContestProvider{
		getUpcomingContestsFn: func(_ context.Context, _ string, _ int) ([]appuser.DashboardContest, error) {
			return []appuser.DashboardContest{
				{ID: "c-1", Name: "Contest A", StartTime: fixedTime.Add(24 * time.Hour), DurationMinutes: 180},
			}, nil
		},
		getActiveContestsFn: func(_ context.Context, _ string, _ int) ([]appuser.DashboardContest, error) {
			return nil, nil
		},
		getFinishedContestResultsFn: func(_ context.Context, _ string, _ int) ([]appuser.DashboardContestResult, error) {
			return []appuser.DashboardContestResult{
				{ContestID: "c-2", ContestName: "Past Contest", Position: 3, ProblemsSolved: 4, Penalty: 120},
			}, nil
		},
	}
	materialProvider := &mockDashboardMaterialProvider{
		getRecentMaterialsFn: func(_ context.Context, _ string, _ int, _ int) ([]appuser.DashboardMaterial, error) {
			return []appuser.DashboardMaterial{
				{ID: "m-1", Title: "Algo Notes", GroupID: "g-1", GroupName: "Team A", PublishedAt: fixedTime, AuthorNickname: "coach1"},
			}, nil
		},
	}
	rankingProvider := &mockDashboardRankingProvider{
		getUserStatsFn: func(_ context.Context, _ string) (int, *int, int, error) {
			return 7, &pos, 100, nil
		},
	}

	uc := appuser.NewGetDashboardUseCase(subProvider, contestProvider, materialProvider, rankingProvider)
	h := newHandlerWithGetDashboard(uc)
	wrapped := wrapWithAuth(
		http.HandlerFunc(h.GetDashboard),
		&domainuser.TokenClaims{UserID: userID, Role: shared.RoleContestant},
	)

	req := httptest.NewRequest(http.MethodGet, "/users/me/dashboard", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}

	var resp dashboardResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(resp.RecentSubmissions) != 1 {
		t.Fatalf("recentSubmissions: want 1, got %d", len(resp.RecentSubmissions))
	}
	s := resp.RecentSubmissions[0]
	if s.ProblemSlug != "two-sum" {
		t.Errorf("submission.problemSlug: want two-sum, got %s", s.ProblemSlug)
	}
	if s.Verdict != "ACCEPTED" {
		t.Errorf("submission.verdict: want ACCEPTED, got %s", s.Verdict)
	}
	if s.ExecutionTime == nil || *s.ExecutionTime != 120 {
		t.Errorf("submission.executionTime: want 120, got %v", s.ExecutionTime)
	}

	if len(resp.UpcomingContests) != 1 {
		t.Fatalf("upcomingContests: want 1, got %d", len(resp.UpcomingContests))
	}
	if resp.UpcomingContests[0].Name != "Contest A" {
		t.Errorf("upcomingContests[0].name: want Contest A, got %s", resp.UpcomingContests[0].Name)
	}

	if resp.ProblemsSolved != 7 {
		t.Errorf("problemsSolved: want 7, got %d", resp.ProblemsSolved)
	}
	if resp.Ranking.Position == nil || *resp.Ranking.Position != 3 {
		t.Errorf("ranking.position: want 3, got %v", resp.Ranking.Position)
	}
	if resp.Ranking.TotalUsers != 100 {
		t.Errorf("ranking.totalUsers: want 100, got %d", resp.Ranking.TotalUsers)
	}

	if len(resp.RecentContestResults) != 1 {
		t.Fatalf("recentContestResults: want 1, got %d", len(resp.RecentContestResults))
	}
	cr := resp.RecentContestResults[0]
	if cr.Position != 3 || cr.ProblemsSolved != 4 || cr.Penalty != 120 {
		t.Errorf("contestResult: want pos=3 solved=4 penalty=120, got pos=%d solved=%d penalty=%d", cr.Position, cr.ProblemsSolved, cr.Penalty)
	}

	if len(resp.RecentMaterials) != 1 || resp.RecentMaterials[0].Title != "Algo Notes" {
		t.Errorf("recentMaterials: unexpected content %+v", resp.RecentMaterials)
	}
}
