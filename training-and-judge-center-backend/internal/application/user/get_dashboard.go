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
	MaterialsCount       int
	CurrentStreak        int
	MaximumStreak        int
	RecentContestResults []DashboardContestResult
}

type GetDashboardUseCase struct {
	submissionProvider     DashboardSubmissionProvider
	contestProvider        DashboardContestProvider
	materialProvider       DashboardMaterialProvider
	problemsSolvedProvider ProblemsSolvedProvider
}

func NewGetDashboardUseCase(
	submissionProvider DashboardSubmissionProvider,
	contestProvider DashboardContestProvider,
	materialProvider DashboardMaterialProvider,
	problemsSolvedProvider ProblemsSolvedProvider,
) *GetDashboardUseCase {
	return &GetDashboardUseCase{
		submissionProvider:     submissionProvider,
		contestProvider:        contestProvider,
		materialProvider:       materialProvider,
		problemsSolvedProvider: problemsSolvedProvider,
	}
}

func (uc *GetDashboardUseCase) Execute(ctx context.Context, in GetDashboardInput) (*GetDashboardOutput, error) {
	userID := in.CurrentUser.ID

	var (
		submissions    []DashboardSubmission
		dates          []time.Time
		upcoming       []DashboardContest
		active         []DashboardContest
		finished       []DashboardContestResult
		problemsSolved int
		materialsCount int
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
		materialsCount, err = uc.materialProvider.GetRecentMaterialsCount(ctx, userID, dashboardRecentMaterialsWindowDays)
		return
	})
	g.Go(func() (err error) {
		problemsSolved, err = uc.problemsSolvedProvider.GetProblemsSolved(ctx, userID)
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
		MaterialsCount:       materialsCount,
		CurrentStreak:        currentStreak,
		MaximumStreak:        maximumStreak,
		RecentContestResults: finished,
	}, nil
}
