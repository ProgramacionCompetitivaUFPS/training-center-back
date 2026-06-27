package user

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
)

const (
	dashboardRecentSubmissionsLimit    = 10
	dashboardUpcomingContestsLimit     = 3
	dashboardActiveContestsLimit       = 3
	dashboardFinishedContestsLimit     = 10
	dashboardRecentMaterialsLimit      = 10
	dashboardRecentMaterialsWindowDays = 30
)

type GetDashboardInput struct {
	CurrentUser appshared.CurrentUser
	Now         time.Time
}

type GetDashboardOutput struct {
	RecentSubmissions    []DashboardSubmission
	UpcomingContests     []DashboardContest
	ActiveContests       []DashboardContest
	ProblemsSolved       int
	RecentMaterials      []DashboardMaterial
	CurrentStreak        int
	MaximumStreak        int
	RankingPosition      *int
	RankingTotalUsers    int
	RecentContestResults []DashboardContestResult
}

type GetDashboardUseCase struct {
	submissionProvider DashboardSubmissionProvider
	contestProvider    DashboardContestProvider
	materialProvider   DashboardMaterialProvider
	rankingProvider    DashboardRankingProvider
}

func NewGetDashboardUseCase(
	submissionProvider DashboardSubmissionProvider,
	contestProvider DashboardContestProvider,
	materialProvider DashboardMaterialProvider,
	rankingProvider DashboardRankingProvider,
) *GetDashboardUseCase {
	return &GetDashboardUseCase{
		submissionProvider: submissionProvider,
		contestProvider:    contestProvider,
		materialProvider:   materialProvider,
		rankingProvider:    rankingProvider,
	}
}

func (uc *GetDashboardUseCase) Execute(ctx context.Context, in GetDashboardInput) (*GetDashboardOutput, error) {
	userID := in.CurrentUser.ID

	var (
		submissions []DashboardSubmission
		dates       []time.Time
		upcoming    []DashboardContest
		active      []DashboardContest
		finished    []DashboardContestResult
		materials   []DashboardMaterial
		problemsSolved, totalUsers int
		position                   *int
	)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() (err error) {
		submissions, err = uc.submissionProvider.GetRecentSubmissions(ctx, userID, dashboardRecentSubmissionsLimit)
		return
	})
	g.Go(func() (err error) {
		dates, err = uc.submissionProvider.GetSubmissionDates(ctx, userID)
		return
	})
	g.Go(func() (err error) {
		upcoming, err = uc.contestProvider.GetUpcomingContests(ctx, userID, dashboardUpcomingContestsLimit)
		return
	})
	g.Go(func() (err error) {
		active, err = uc.contestProvider.GetActiveContests(ctx, userID, dashboardActiveContestsLimit)
		return
	})
	g.Go(func() (err error) {
		finished, err = uc.contestProvider.GetFinishedContestResults(ctx, userID, dashboardFinishedContestsLimit)
		return
	})
	g.Go(func() (err error) {
		materials, err = uc.materialProvider.GetRecentMaterials(ctx, userID, dashboardRecentMaterialsLimit, dashboardRecentMaterialsWindowDays)
		return
	})
	g.Go(func() (err error) {
		problemsSolved, position, totalUsers, err = uc.rankingProvider.GetUserStats(ctx, userID)
		return
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	currentStreak, maximumStreak := computeStreak(dates, in.Now)

	return &GetDashboardOutput{
		RecentSubmissions:    submissions,
		UpcomingContests:     upcoming,
		ActiveContests:       active,
		ProblemsSolved:       problemsSolved,
		RecentMaterials:      materials,
		CurrentStreak:        currentStreak,
		MaximumStreak:        maximumStreak,
		RankingPosition:      position,
		RankingTotalUsers:    totalUsers,
		RecentContestResults: finished,
	}, nil
}
