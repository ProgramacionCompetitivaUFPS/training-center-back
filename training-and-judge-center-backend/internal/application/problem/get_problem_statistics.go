package problem

import (
	"context"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GetProblemStatisticsInput struct {
	Slug string
}

type GetProblemStatisticsOutput struct {
	HasSubmissions   bool
	TotalSubmissions int
	UniqueAttempted  int
	UniqueSolved     int
	ByLanguage       []LanguageStat
	ByVerdict        []VerdictStat
}

type GetProblemStatisticsUseCase struct {
	repo          domainProblem.Repository
	statsProvider StatisticsProvider
}

func NewGetProblemStatisticsUseCase(repo domainProblem.Repository, statsProvider StatisticsProvider) *GetProblemStatisticsUseCase {
	return &GetProblemStatisticsUseCase{repo: repo, statsProvider: statsProvider}
}

func (uc *GetProblemStatisticsUseCase) Execute(ctx context.Context, in GetProblemStatisticsInput) (*GetProblemStatisticsOutput, error) {
	slug, err := domainProblem.NewSlug(in.Slug)
	if err != nil {
		return nil, err
	}

	p, err := uc.repo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	if !p.Status().IsPublished() {
		return nil, apperror.NewForbidden(ErrCodeProblemNotPublished, "Statistics are only available for published problems")
	}

	stats, err := uc.statsProvider.GetByProblemID(ctx, p.ID())
	if err != nil {
		return nil, err
	}

	if stats.TotalSubmissions == 0 {
		return &GetProblemStatisticsOutput{HasSubmissions: false, TotalSubmissions: 0}, nil
	}

	return &GetProblemStatisticsOutput{
		HasSubmissions:   true,
		TotalSubmissions: stats.TotalSubmissions,
		UniqueAttempted:  stats.UniqueAttempted,
		UniqueSolved:     stats.UniqueSolved,
		ByLanguage:       stats.ByLanguage,
		ByVerdict:        stats.ByVerdict,
	}, nil
}
