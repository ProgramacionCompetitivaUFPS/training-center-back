package problem

import (
	"context"
	"errors"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ── StatisticsProvider mock ──────────────────────────────────────────────────

type mockStatisticsProvider struct {
	stats *ProblemStats
	err   error
}

func (m *mockStatisticsProvider) GetByProblemID(_ context.Context, _ string) (*ProblemStats, error) {
	return m.stats, m.err
}

func defaultStats() *ProblemStats {
	return &ProblemStats{
		TotalSubmissions: 10,
		UniqueAttempted:  5,
		UniqueSolved:     3,
		ByLanguage: []LanguageStat{
			{Language: "cpp20", UsersAccepted: 3, UsersAttempted: 5},
		},
		ByVerdict: []VerdictStat{
			{Verdict: "ACCEPTED", Count: 4},
			{Verdict: "WRONG_ANSWER", Count: 6},
		},
	}
}

func newStatisticsUC(repo domainProblem.Repository, provider StatisticsProvider) *GetProblemStatisticsUseCase {
	return NewGetProblemStatisticsUseCase(repo, provider)
}

// ── Tests ────────────────────────────────────────────────────────────────────

func TestGetProblemStatistics_NotFound(t *testing.T) {
	repo := &mockProblemRepository{
		findBySlugFn: func(_ context.Context, _ domainProblem.Slug) (*domainProblem.Problem, error) {
			return nil, apperror.NewNotFound(domainProblem.ErrCodeProblemNotFound, "not found")
		},
	}
	uc := newStatisticsUC(repo, &mockStatisticsProvider{})
	_, err := uc.Execute(context.Background(), GetProblemStatisticsInput{Slug: testSlug})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainProblem.ErrCodeProblemNotFound {
		t.Errorf("expected PROBLEM_NOT_FOUND, got %v", err)
	}
}

func TestGetProblemStatistics_DraftForbidden(t *testing.T) {
	uc := newStatisticsUC(repoWith(newDraftProblem()), &mockStatisticsProvider{})
	_, err := uc.Execute(context.Background(), GetProblemStatisticsInput{Slug: testSlug})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeProblemNotPublished {
		t.Errorf("expected PROBLEM_NOT_PUBLISHED, got %v", err)
	}
}

func TestGetProblemStatistics_DraftForbiddenEvenAsAdmin(t *testing.T) {
	uc := newStatisticsUC(repoWith(newDraftProblemWithModifier()), &mockStatisticsProvider{})
	_, err := uc.Execute(context.Background(), GetProblemStatisticsInput{Slug: testSlug})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeProblemNotPublished {
		t.Errorf("expected PROBLEM_NOT_PUBLISHED, got %v", err)
	}
}

func TestGetProblemStatistics_NoSubmissions(t *testing.T) {
	provider := &mockStatisticsProvider{stats: &ProblemStats{TotalSubmissions: 0}}
	uc := newStatisticsUC(repoWith(newPublishedProblem()), provider)

	out, err := uc.Execute(context.Background(), GetProblemStatisticsInput{Slug: testSlug})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.HasSubmissions {
		t.Error("expected HasSubmissions=false")
	}
	if out.TotalSubmissions != 0 {
		t.Errorf("expected TotalSubmissions=0, got %d", out.TotalSubmissions)
	}
}

func TestGetProblemStatistics_WithSubmissions(t *testing.T) {
	stats := defaultStats()
	provider := &mockStatisticsProvider{stats: stats}
	uc := newStatisticsUC(repoWith(newPublishedProblem()), provider)

	out, err := uc.Execute(context.Background(), GetProblemStatisticsInput{Slug: testSlug})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.HasSubmissions {
		t.Error("expected HasSubmissions=true")
	}
	if out.TotalSubmissions != stats.TotalSubmissions {
		t.Errorf("expected TotalSubmissions=%d, got %d", stats.TotalSubmissions, out.TotalSubmissions)
	}
	if out.UniqueAttempted != stats.UniqueAttempted {
		t.Errorf("expected UniqueAttempted=%d, got %d", stats.UniqueAttempted, out.UniqueAttempted)
	}
	if out.UniqueSolved != stats.UniqueSolved {
		t.Errorf("expected UniqueSolved=%d, got %d", stats.UniqueSolved, out.UniqueSolved)
	}
	if len(out.ByLanguage) != len(stats.ByLanguage) {
		t.Errorf("expected %d language stats, got %d", len(stats.ByLanguage), len(out.ByLanguage))
	}
	if len(out.ByVerdict) != len(stats.ByVerdict) {
		t.Errorf("expected %d verdict stats, got %d", len(stats.ByVerdict), len(out.ByVerdict))
	}
}

func TestGetProblemStatistics_InvalidSlug(t *testing.T) {
	uc := newStatisticsUC(&mockProblemRepository{}, &mockStatisticsProvider{})
	_, err := uc.Execute(context.Background(), GetProblemStatisticsInput{Slug: "AB"})
	if err == nil {
		t.Fatal("expected error for invalid slug, got nil")
	}
}
