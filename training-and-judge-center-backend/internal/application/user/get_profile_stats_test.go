package user

import (
	"context"
	"errors"
	"testing"
)

// ── mock providers ────────────────────────────────────────────────────────────

type mockRankingProvider struct {
	getRankingFn func(ctx context.Context, userID string) (int, *int, int, error)
}

func (m *mockRankingProvider) GetRanking(ctx context.Context, userID string) (int, *int, int, error) {
	if m.getRankingFn != nil {
		return m.getRankingFn(ctx, userID)
	}
	return 0, nil, 0, nil
}

type mockSubmissionStatsProvider struct {
	getSubmissionCountsFn func(ctx context.Context, userID string) (int, int, error)
}

func (m *mockSubmissionStatsProvider) GetSubmissionCounts(ctx context.Context, userID string) (int, int, error) {
	if m.getSubmissionCountsFn != nil {
		return m.getSubmissionCountsFn(ctx, userID)
	}
	return 0, 0, nil
}

type mockContestParticipationProvider struct {
	getContestsParticipatedCountFn func(ctx context.Context, userID string) (int, error)
}

func (m *mockContestParticipationProvider) GetContestsParticipatedCount(ctx context.Context, userID string) (int, error) {
	if m.getContestsParticipatedCountFn != nil {
		return m.getContestsParticipatedCountFn(ctx, userID)
	}
	return 0, nil
}

type mockTopicStatsProvider struct {
	getTopicBreakdownFn func(ctx context.Context, userID string) ([]TopicStat, error)
}

func (m *mockTopicStatsProvider) GetTopicBreakdown(ctx context.Context, userID string) ([]TopicStat, error) {
	if m.getTopicBreakdownFn != nil {
		return m.getTopicBreakdownFn(ctx, userID)
	}
	return nil, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func defaultStatsProviders() (*mockRankingProvider, *mockSubmissionStatsProvider, *mockContestParticipationProvider, *mockTopicStatsProvider) {
	return &mockRankingProvider{}, &mockSubmissionStatsProvider{}, &mockContestParticipationProvider{}, &mockTopicStatsProvider{}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestGetProfileStats_EmptyProviders_ReturnsZeroValues(t *testing.T) {
	rank, sub, con, topic := defaultStatsProviders()
	uc := NewGetProfileStatsUseCase(rank, sub, con, topic)

	out, err := uc.Execute(context.Background(), GetProfileStatsInput{CurrentUser: currentUser("u-1")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.ProblemsSolved != 0 {
		t.Errorf("want 0 problems solved, got %d", out.ProblemsSolved)
	}
	if out.TotalSubmissions != 0 || out.AcceptedSubmissions != 0 {
		t.Errorf("want 0 submissions, got total=%d accepted=%d", out.TotalSubmissions, out.AcceptedSubmissions)
	}
	if out.ContestsParticipated != 0 {
		t.Errorf("want 0 contests participated, got %d", out.ContestsParticipated)
	}
	if out.RankingPosition != nil {
		t.Errorf("want nil ranking position, got %v", out.RankingPosition)
	}
	if len(out.TopicStats) != 0 {
		t.Errorf("want empty topic stats, got %+v", out.TopicStats)
	}
}

func TestGetProfileStats_RankingProviderError_Propagates(t *testing.T) {
	boom := errors.New("ranking error")
	rank := &mockRankingProvider{
		getRankingFn: func(_ context.Context, _ string) (int, *int, int, error) {
			return 0, nil, 0, boom
		},
	}
	_, sub, con, topic := defaultStatsProviders()
	uc := NewGetProfileStatsUseCase(rank, sub, con, topic)

	_, err := uc.Execute(context.Background(), GetProfileStatsInput{CurrentUser: currentUser("u-1")})
	if !errors.Is(err, boom) {
		t.Errorf("expected ranking error, got %v", err)
	}
}

func TestGetProfileStats_SubmissionStatsProviderError_Propagates(t *testing.T) {
	boom := errors.New("submission stats error")
	sub := &mockSubmissionStatsProvider{
		getSubmissionCountsFn: func(_ context.Context, _ string) (int, int, error) {
			return 0, 0, boom
		},
	}
	rank, _, con, topic := defaultStatsProviders()
	uc := NewGetProfileStatsUseCase(rank, sub, con, topic)

	_, err := uc.Execute(context.Background(), GetProfileStatsInput{CurrentUser: currentUser("u-1")})
	if !errors.Is(err, boom) {
		t.Errorf("expected submission stats error, got %v", err)
	}
}

func TestGetProfileStats_ContestParticipationProviderError_Propagates(t *testing.T) {
	boom := errors.New("contest participation error")
	con := &mockContestParticipationProvider{
		getContestsParticipatedCountFn: func(_ context.Context, _ string) (int, error) {
			return 0, boom
		},
	}
	rank, sub, _, topic := defaultStatsProviders()
	uc := NewGetProfileStatsUseCase(rank, sub, con, topic)

	_, err := uc.Execute(context.Background(), GetProfileStatsInput{CurrentUser: currentUser("u-1")})
	if !errors.Is(err, boom) {
		t.Errorf("expected contest participation error, got %v", err)
	}
}

func TestGetProfileStats_TopicStatsProviderError_Propagates(t *testing.T) {
	boom := errors.New("topic stats error")
	topic := &mockTopicStatsProvider{
		getTopicBreakdownFn: func(_ context.Context, _ string) ([]TopicStat, error) {
			return nil, boom
		},
	}
	rank, sub, con, _ := defaultStatsProviders()
	uc := NewGetProfileStatsUseCase(rank, sub, con, topic)

	_, err := uc.Execute(context.Background(), GetProfileStatsInput{CurrentUser: currentUser("u-1")})
	if !errors.Is(err, boom) {
		t.Errorf("expected topic stats error, got %v", err)
	}
}

func TestGetProfileStats_PopulatedData_MapsToOutput(t *testing.T) {
	pos := 5

	rank := &mockRankingProvider{
		getRankingFn: func(_ context.Context, _ string) (int, *int, int, error) {
			return 10, &pos, 50, nil
		},
	}
	sub := &mockSubmissionStatsProvider{
		getSubmissionCountsFn: func(_ context.Context, _ string) (int, int, error) {
			return 132, 58, nil
		},
	}
	con := &mockContestParticipationProvider{
		getContestsParticipatedCountFn: func(_ context.Context, _ string) (int, error) {
			return 9, nil
		},
	}
	topic := &mockTopicStatsProvider{
		getTopicBreakdownFn: func(_ context.Context, _ string) ([]TopicStat, error) {
			return []TopicStat{{Tag: "graphs", Solved: 15}, {Tag: "dp", Solved: 12}}, nil
		},
	}

	uc := NewGetProfileStatsUseCase(rank, sub, con, topic)
	out, err := uc.Execute(context.Background(), GetProfileStatsInput{CurrentUser: currentUser("u-99")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.ProblemsSolved != 10 {
		t.Errorf("ProblemsSolved: want 10, got %d", out.ProblemsSolved)
	}
	if out.TotalSubmissions != 132 || out.AcceptedSubmissions != 58 {
		t.Errorf("submission counts: want total=132 accepted=58, got total=%d accepted=%d", out.TotalSubmissions, out.AcceptedSubmissions)
	}
	if out.ContestsParticipated != 9 {
		t.Errorf("ContestsParticipated: want 9, got %d", out.ContestsParticipated)
	}
	if out.RankingPosition == nil || *out.RankingPosition != 5 {
		t.Errorf("RankingPosition: want 5, got %v", out.RankingPosition)
	}
	if out.RankingTotalUsers != 50 {
		t.Errorf("RankingTotalUsers: want 50, got %d", out.RankingTotalUsers)
	}
	if len(out.TopicStats) != 2 || out.TopicStats[0].Tag != "graphs" {
		t.Errorf("TopicStats unexpected: %+v", out.TopicStats)
	}
}
