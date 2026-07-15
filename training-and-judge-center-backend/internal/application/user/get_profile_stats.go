package user

import (
	"context"

	"golang.org/x/sync/errgroup"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
)

type GetProfileStatsInput struct {
	CurrentUser appshared.CurrentUser
}

type GetProfileStatsOutput struct {
	ProblemsSolved       int
	TotalSubmissions     int
	AcceptedSubmissions  int
	ContestsParticipated int
	RankingPosition      *int
	RankingTotalUsers    int
	TopicStats           []TopicStat
}

type GetProfileStatsUseCase struct {
	rankingProvider    RankingProvider
	submissionProvider SubmissionStatsProvider
	contestProvider    ContestParticipationProvider
	topicProvider      TopicStatsProvider
}

func NewGetProfileStatsUseCase(
	rankingProvider RankingProvider,
	submissionProvider SubmissionStatsProvider,
	contestProvider ContestParticipationProvider,
	topicProvider TopicStatsProvider,
) *GetProfileStatsUseCase {
	return &GetProfileStatsUseCase{
		rankingProvider:    rankingProvider,
		submissionProvider: submissionProvider,
		contestProvider:    contestProvider,
		topicProvider:      topicProvider,
	}
}

func (uc *GetProfileStatsUseCase) Execute(ctx context.Context, in GetProfileStatsInput) (*GetProfileStatsOutput, error) {
	userID := in.CurrentUser.ID

	var (
		problemsSolved, totalUsers int
		position                   *int
		totalSubmissions, accepted int
		contestsParticipated       int
		topicStats                 []TopicStat
	)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() (err error) {
		problemsSolved, position, totalUsers, err = uc.rankingProvider.GetRanking(ctx, userID)
		return
	})
	g.Go(func() (err error) {
		totalSubmissions, accepted, err = uc.submissionProvider.GetSubmissionCounts(ctx, userID)
		return
	})
	g.Go(func() (err error) {
		contestsParticipated, err = uc.contestProvider.GetContestsParticipatedCount(ctx, userID)
		return
	})
	g.Go(func() (err error) {
		topicStats, err = uc.topicProvider.GetTopicBreakdown(ctx, userID)
		return
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &GetProfileStatsOutput{
		ProblemsSolved:       problemsSolved,
		TotalSubmissions:     totalSubmissions,
		AcceptedSubmissions:  accepted,
		ContestsParticipated: contestsParticipated,
		RankingPosition:      position,
		RankingTotalUsers:    totalUsers,
		TopicStats:           topicStats,
	}, nil
}
