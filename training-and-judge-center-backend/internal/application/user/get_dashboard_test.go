package user

import (
	"context"
	"errors"
	"testing"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

// ── mock providers ────────────────────────────────────────────────────────────

type mockSubmissionProvider struct {
	getRecentFn func(ctx context.Context, userID string, limit int) ([]DashboardSubmission, error)
	getDatesFn  func(ctx context.Context, userID string) ([]time.Time, error)
}

func (m *mockSubmissionProvider) GetRecentSubmissions(ctx context.Context, userID string, limit int) ([]DashboardSubmission, error) {
	if m.getRecentFn != nil {
		return m.getRecentFn(ctx, userID, limit)
	}
	return nil, nil
}

func (m *mockSubmissionProvider) GetSubmissionDates(ctx context.Context, userID string) ([]time.Time, error) {
	if m.getDatesFn != nil {
		return m.getDatesFn(ctx, userID)
	}
	return nil, nil
}

type mockContestProvider struct {
	getUpcomingFn  func(ctx context.Context, userID string, limit int) ([]DashboardContest, error)
	getActiveFn    func(ctx context.Context, userID string, limit int) ([]DashboardContest, error)
	getFinishedFn  func(ctx context.Context, userID string, limit int) ([]DashboardContestResult, error)
}

func (m *mockContestProvider) GetUpcomingContests(ctx context.Context, userID string, limit int) ([]DashboardContest, error) {
	if m.getUpcomingFn != nil {
		return m.getUpcomingFn(ctx, userID, limit)
	}
	return nil, nil
}

func (m *mockContestProvider) GetActiveContests(ctx context.Context, userID string, limit int) ([]DashboardContest, error) {
	if m.getActiveFn != nil {
		return m.getActiveFn(ctx, userID, limit)
	}
	return nil, nil
}

func (m *mockContestProvider) GetFinishedContestResults(ctx context.Context, userID string, limit int) ([]DashboardContestResult, error) {
	if m.getFinishedFn != nil {
		return m.getFinishedFn(ctx, userID, limit)
	}
	return nil, nil
}

type mockMaterialProvider struct {
	getRecentFn func(ctx context.Context, userID string, limit int, windowDays int) ([]DashboardMaterial, error)
}

func (m *mockMaterialProvider) GetRecentMaterials(ctx context.Context, userID string, limit int, windowDays int) ([]DashboardMaterial, error) {
	if m.getRecentFn != nil {
		return m.getRecentFn(ctx, userID, limit, windowDays)
	}
	return nil, nil
}

type mockRankingProvider struct {
	getStatsFn func(ctx context.Context, userID string) (int, *int, int, error)
}

func (m *mockRankingProvider) GetUserStats(ctx context.Context, userID string) (int, *int, int, error) {
	if m.getStatsFn != nil {
		return m.getStatsFn(ctx, userID)
	}
	return 0, nil, 0, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func intPtr(i int) *int { return &i }

func currentUser(id string) appshared.CurrentUser {
	return appshared.CurrentUser{ID: id, Role: shared.RoleContestant}
}

func defaultProviders() (*mockSubmissionProvider, *mockContestProvider, *mockMaterialProvider, *mockRankingProvider) {
	return &mockSubmissionProvider{}, &mockContestProvider{}, &mockMaterialProvider{}, &mockRankingProvider{}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestGetDashboard_EmptyProviders_ReturnsZeroValues(t *testing.T) {
	sub, con, mat, rank := defaultProviders()
	uc := NewGetDashboardUseCase(sub, con, mat, rank)

	out, err := uc.Execute(context.Background(), GetDashboardInput{CurrentUser: currentUser("u-1")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(out.RecentSubmissions) != 0 {
		t.Errorf("want 0 submissions, got %d", len(out.RecentSubmissions))
	}
	if out.ProblemsSolved != 0 {
		t.Errorf("want 0 problems solved, got %d", out.ProblemsSolved)
	}
	if out.RankingPosition != nil {
		t.Errorf("want nil position, got %v", out.RankingPosition)
	}
	if out.CurrentStreak != 0 || out.MaximumStreak != 0 {
		t.Errorf("want streak (0,0), got (%d,%d)", out.CurrentStreak, out.MaximumStreak)
	}
}

func TestGetDashboard_SubmissionProviderError_Propagates(t *testing.T) {
	boom := errors.New("db error")
	sub := &mockSubmissionProvider{
		getRecentFn: func(_ context.Context, _ string, _ int) ([]DashboardSubmission, error) {
			return nil, boom
		},
	}
	_, con, mat, rank := defaultProviders()
	uc := NewGetDashboardUseCase(sub, con, mat, rank)

	_, err := uc.Execute(context.Background(), GetDashboardInput{CurrentUser: currentUser("u-1")})
	if !errors.Is(err, boom) {
		t.Errorf("expected db error, got %v", err)
	}
}

func TestGetDashboard_ContestProviderError_Propagates(t *testing.T) {
	boom := errors.New("contest db error")
	con := &mockContestProvider{
		getUpcomingFn: func(_ context.Context, _ string, _ int) ([]DashboardContest, error) {
			return nil, boom
		},
	}
	sub, _, mat, rank := defaultProviders()
	uc := NewGetDashboardUseCase(sub, con, mat, rank)

	_, err := uc.Execute(context.Background(), GetDashboardInput{CurrentUser: currentUser("u-1")})
	if !errors.Is(err, boom) {
		t.Errorf("expected contest db error, got %v", err)
	}
}

func TestGetDashboard_RankingProviderError_Propagates(t *testing.T) {
	boom := errors.New("ranking error")
	rank := &mockRankingProvider{
		getStatsFn: func(_ context.Context, _ string) (int, *int, int, error) {
			return 0, nil, 0, boom
		},
	}
	sub, con, mat, _ := defaultProviders()
	uc := NewGetDashboardUseCase(sub, con, mat, rank)

	_, err := uc.Execute(context.Background(), GetDashboardInput{CurrentUser: currentUser("u-1")})
	if !errors.Is(err, boom) {
		t.Errorf("expected ranking error, got %v", err)
	}
}

func TestGetDashboard_PopulatedData_MapsToOutput(t *testing.T) {
	execMs := 200
	memKb := 8192
	pos := 5
	fixedTime := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)

	sub := &mockSubmissionProvider{
		getRecentFn: func(_ context.Context, _ string, _ int) ([]DashboardSubmission, error) {
			return []DashboardSubmission{
				{ID: "s-1", ProblemSlug: "binary-search", Verdict: "ACCEPTED", Language: "go",
					SubmittedAt: fixedTime, ExecutionTime: &execMs, MemoryKb: &memKb},
			}, nil
		},
		getDatesFn: func(_ context.Context, _ string) ([]time.Time, error) {
			return []time.Time{fixedTime}, nil
		},
	}
	con := &mockContestProvider{
		getUpcomingFn: func(_ context.Context, _ string, _ int) ([]DashboardContest, error) {
			return []DashboardContest{{ID: "c-1", Name: "Round 1", StartTime: fixedTime.Add(48 * time.Hour), DurationMinutes: 120}}, nil
		},
		getActiveFn: func(_ context.Context, _ string, _ int) ([]DashboardContest, error) {
			return nil, nil
		},
		getFinishedFn: func(_ context.Context, _ string, _ int) ([]DashboardContestResult, error) {
			return []DashboardContestResult{{ContestID: "c-0", ContestName: "Warmup", Position: 1, ProblemsSolved: 3, Penalty: 60}}, nil
		},
	}
	mat := &mockMaterialProvider{
		getRecentFn: func(_ context.Context, _ string, _ int, _ int) ([]DashboardMaterial, error) {
			return []DashboardMaterial{{ID: "m-1", Title: "Graphs", GroupID: "g-1", GroupName: "G1", PublishedAt: fixedTime, AuthorNickname: "admin"}}, nil
		},
	}
	rank := &mockRankingProvider{
		getStatsFn: func(_ context.Context, _ string) (int, *int, int, error) {
			return 10, &pos, 50, nil
		},
	}

	uc := NewGetDashboardUseCase(sub, con, mat, rank)
	out, err := uc.Execute(context.Background(), GetDashboardInput{CurrentUser: currentUser("u-99")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(out.RecentSubmissions) != 1 || out.RecentSubmissions[0].ProblemSlug != "binary-search" {
		t.Errorf("RecentSubmissions unexpected: %+v", out.RecentSubmissions)
	}
	if len(out.UpcomingContests) != 1 || out.UpcomingContests[0].Name != "Round 1" {
		t.Errorf("UpcomingContests unexpected: %+v", out.UpcomingContests)
	}
	if len(out.RecentContestResults) != 1 || out.RecentContestResults[0].Position != 1 {
		t.Errorf("RecentContestResults unexpected: %+v", out.RecentContestResults)
	}
	if out.ProblemsSolved != 10 {
		t.Errorf("ProblemsSolved: want 10, got %d", out.ProblemsSolved)
	}
	if out.RankingPosition == nil || *out.RankingPosition != 5 {
		t.Errorf("RankingPosition: want 5, got %v", out.RankingPosition)
	}
	if out.RankingTotalUsers != 50 {
		t.Errorf("RankingTotalUsers: want 50, got %d", out.RankingTotalUsers)
	}
	if len(out.RecentMaterials) != 1 || out.RecentMaterials[0].Title != "Graphs" {
		t.Errorf("RecentMaterials unexpected: %+v", out.RecentMaterials)
	}
}

func TestGetDashboard_StreakComputedFromDates(t *testing.T) {
	// Two consecutive days ending on fixedTime — current streak = 2.
	fixedTime := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	yesterday := fixedTime.Add(-24 * time.Hour)

	sub := &mockSubmissionProvider{
		getDatesFn: func(_ context.Context, _ string) ([]time.Time, error) {
			// Return UTC midnight dates as computeStreak expects
			d1 := yesterday.UTC().Truncate(24 * time.Hour)
			d2 := fixedTime.UTC().Truncate(24 * time.Hour)
			return []time.Time{d1, d2}, nil
		},
	}
	_, con, mat, rank := defaultProviders()
	uc := NewGetDashboardUseCase(sub, con, mat, rank)

	out, err := uc.Execute(context.Background(), GetDashboardInput{CurrentUser: currentUser("u-streak")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.CurrentStreak != 2 {
		t.Errorf("CurrentStreak: want 2, got %d", out.CurrentStreak)
	}
	if out.MaximumStreak != 2 {
		t.Errorf("MaximumStreak: want 2, got %d", out.MaximumStreak)
	}
}
