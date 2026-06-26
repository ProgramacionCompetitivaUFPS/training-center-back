package user

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
)

const (
	dashboardRecentSubmissionsLimit  = 10
	dashboardUpcomingContestsLimit   = 3
	dashboardActiveContestsLimit     = 3
	dashboardFinishedContestsLimit   = 10
	dashboardRecentMaterialsLimit    = 10
	dashboardRecentMaterialsWindowDays = 30
)

type GetDashboardInput struct {
	CurrentUser appshared.CurrentUser
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

	submissions, err := uc.submissionProvider.GetRecentSubmissions(ctx, userID, dashboardRecentSubmissionsLimit)
	if err != nil {
		return nil, err
	}

	dates, err := uc.submissionProvider.GetSubmissionDates(ctx, userID)
	if err != nil {
		return nil, err
	}

	upcoming, err := uc.contestProvider.GetUpcomingContests(ctx, userID, dashboardUpcomingContestsLimit)
	if err != nil {
		return nil, err
	}

	active, err := uc.contestProvider.GetActiveContests(ctx, userID, dashboardActiveContestsLimit)
	if err != nil {
		return nil, err
	}

	finished, err := uc.contestProvider.GetFinishedContestResults(ctx, userID, dashboardFinishedContestsLimit)
	if err != nil {
		return nil, err
	}

	materials, err := uc.materialProvider.GetRecentMaterials(ctx, userID, dashboardRecentMaterialsLimit, dashboardRecentMaterialsWindowDays)
	if err != nil {
		return nil, err
	}

	problemsSolved, position, totalUsers, err := uc.rankingProvider.GetUserStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	currentStreak, maximumStreak := computeStreak(dates, time.Now())

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
