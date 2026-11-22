//go:build integration

package contest

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appContest "github.com/training-judge-center/backend/internal/application/contest"
	"github.com/training-judge-center/backend/internal/testutil/pgtest"
)

func itRegister(t *testing.T, pool *pgxpool.Pool, contestID, userID string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `INSERT INTO contest_registrations (contest_id, user_id) VALUES ($1, $2)`, contestID, userID)
	require.NoError(t, err)
}

func itSubmissionIDs(subs []appContest.RichSubmissionData) []string {
	out := make([]string, len(subs))
	for i, s := range subs {
		out[i] = s.ID
	}
	return out
}

func itSeedContestSubmissions(t *testing.T, pool *pgxpool.Pool) time.Time {
	t.Helper()
	itSeedWithContests(t, pool)
	itSeedTeams(t, pool)
	pgtest.SeedContestProblem(t, pool, "dddddddd-0000-0000-0000-000000000001", itContest1, itProblem1, 1)
	pgtest.SeedContestProblem(t, pool, "dddddddd-0000-0000-0000-000000000002", itContest1, itProblem2, 2)
	itSeedTeamEntry(t, pool, itPart1, itContest1, itTeam1, itOwner, itUserC)
	t0 := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	sub := func(id, problem, user, standing string, at time.Time) {
		pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: id, ProblemID: problem, UserID: user, ContestID: itContest1, StandingID: standing, SubmittedAt: at})
	}
	sub("s1", itProblem1, itOwner, itOwner, t0)
	sub("s2", itProblem2, itUserB, itUserB, t0.Add(time.Hour))
	sub("s3", itProblem1, itUserC, itTeam1, t0.Add(2*time.Hour))
	sub("s4", itProblem1, itUserB, itUserB, t0.Add(3*time.Hour))
	return t0
}

func TestContestSubmissionProvider_ListByContest_ShapesRowsWithProblemOrderAndTeam(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedContestSubmissions(t, pool)

	got, total, err := NewContestSubmissionProvider(pool).ListByContest(context.Background(), itContest1, appContest.ContestSubmissionFilters{Limit: 10})

	require.NoError(t, err)
	assert.Equal(t, 4, total)
	assert.Equal(t, []string{"s4", "s3", "s2", "s1"}, itSubmissionIDs(got))
	team := got[1]
	assert.Equal(t, itUserC, team.UserID)
	assert.Equal(t, itTeam1, team.StandingID)
	require.NotNil(t, team.TeamID)
	assert.Equal(t, itTeam1, *team.TeamID)
	require.NotNil(t, team.TeamName)
	assert.Equal(t, "Team One", *team.TeamName)
	assert.Equal(t, []string{"carla", "owner"}, team.TeamMemberNicknames)
	assert.Equal(t, 1, team.ProblemOrder)
	assert.Equal(t, "two-sum", team.ProblemSlug)
	assert.Equal(t, "carla", team.Nickname)
	individual := got[2]
	assert.Equal(t, 2, individual.ProblemOrder)
	assert.Nil(t, individual.TeamID)
	assert.Empty(t, individual.TeamMemberNicknames)
	assert.Equal(t, "PRIVATE", individual.Visibility)
	assert.False(t, individual.IsBlockly)
}

func TestContestSubmissionProvider_ListByContest_Filters(t *testing.T) {
	pool := pgtest.NewDB(t)
	t0 := itSeedContestSubmissions(t, pool)
	provider := NewContestSubmissionProvider(pool)
	slug, beto, owner, carla := "two-sum", "beto", "owner", "carla"
	atMost, after, freeze := t0.Add(time.Hour), t0.Add(time.Hour), t0.Add(90*time.Minute)

	tests := []struct {
		name      string
		filters   appContest.ContestSubmissionFilters
		wantIDs   []string
		wantTotal int
	}{
		{"by problem slug", appContest.ContestSubmissionFilters{ProblemSlug: &slug}, []string{"s4", "s3", "s1"}, 3},
		{"by individual nickname", appContest.ContestSubmissionFilters{Nickname: &beto}, []string{"s4", "s2"}, 2},
		{"nickname of a selected team member matches the team's submissions", appContest.ContestSubmissionFilters{Nickname: &owner}, []string{"s3", "s1"}, 2},
		{"nickname of the team member who submitted", appContest.ContestSubmissionFilters{Nickname: &carla}, []string{"s3"}, 1},
		{"submitted at most", appContest.ContestSubmissionFilters{SubmittedAtMost: &atMost}, []string{"s2", "s1"}, 2},
		{"submitted after", appContest.ContestSubmissionFilters{SubmittedAfter: &after}, []string{"s4", "s3"}, 2},
		{"freeze hides later rows except the caller's standing", appContest.ContestSubmissionFilters{FreezeAt: &freeze, ExceptStandingID: itUserB}, []string{"s4", "s2", "s1"}, 3},
		{"freeze with no standing hides every later row", appContest.ContestSubmissionFilters{FreezeAt: &freeze}, []string{"s2", "s1"}, 2},
		{"second page", appContest.ContestSubmissionFilters{Limit: 2, Offset: 2}, []string{"s2", "s1"}, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.filters.Limit == 0 {
				tt.filters.Limit = 10
			}

			got, total, err := provider.ListByContest(context.Background(), itContest1, tt.filters)

			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, total)
			assert.Equal(t, tt.wantIDs, itSubmissionIDs(got))
		})
	}
}

func TestContestSubmissionProvider_ListByContest_OffsetPastTheEndReturnsOnlyTheTotal(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedContestSubmissions(t, pool)

	got, total, err := NewContestSubmissionProvider(pool).ListByContest(context.Background(), itContest1, appContest.ContestSubmissionFilters{Limit: 10, Offset: 4})

	require.NoError(t, err)
	assert.Equal(t, 4, total)
	assert.Empty(t, got)
}

func TestDashboardProvider_UpcomingAndActiveContests_OnlyTheOnesTheUserTakesPartIn(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	pgtest.SeedUser(t, pool, itUserC, "carla")
	itSeedTeams(t, pool)
	now := time.Now()
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: itContest1, GroupID: itGroupA, OwnerID: itOwner, Name: "Upcoming Mine", Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour)})
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: itContest2, GroupID: itGroupB, OwnerID: itOwner, Name: "Upcoming Not Mine", Start: now.Add(time.Hour), End: now.Add(2 * time.Hour)})
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: itContest3, GroupID: itGroupA, OwnerID: itOwner, Name: "Active Via Team", Start: now.Add(-time.Hour), End: now.Add(time.Hour)})
	itRegister(t, pool, itContest1, itOwner)
	itSeedTeamEntry(t, pool, itPart1, itContest3, itTeam1, itOwner)
	provider := NewDashboardProvider(pool)

	upcoming, err := provider.GetUpcomingContests(context.Background(), itOwner, 10)
	require.NoError(t, err)
	require.Len(t, upcoming, 1)
	assert.Equal(t, "Upcoming Mine", upcoming[0].Name)
	assert.Equal(t, 60, upcoming[0].DurationMinutes)
	assert.Equal(t, itGroupA, upcoming[0].GroupID)
	assert.Equal(t, "Group A", upcoming[0].GroupName)

	active, err := provider.GetActiveContests(context.Background(), itOwner, 10)
	require.NoError(t, err)
	require.Len(t, active, 1)
	assert.Equal(t, "Active Via Team", active[0].Name)

	none, err := provider.GetUpcomingContests(context.Background(), itUserC, 10)
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestDashboardProvider_GetFinishedContestResults_ScoresSolvedProblemsAndPenalty(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	now := time.Now().UTC()
	start := now.Add(-5 * time.Hour)
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: itContest1, GroupID: itGroupA, OwnerID: itOwner, Name: "Finished", Start: start, End: now.Add(-3 * time.Hour)})
	itRegister(t, pool, itContest1, itOwner)
	sub := func(id, problem, user, status string, minute int) {
		pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: id, ProblemID: problem, UserID: user, ContestID: itContest1, Status: status, SubmittedAt: start.Add(time.Duration(minute) * time.Minute)})
	}
	sub("wa", itProblem1, itOwner, "WRONG_ANSWER", 10)
	sub("ac1", itProblem1, itOwner, "ACCEPTED", 30)
	sub("ac2", itProblem2, itOwner, "ACCEPTED", 50)

	got, err := NewDashboardProvider(pool).GetFinishedContestResults(context.Background(), itOwner, 5)

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, itContest1, got[0].ContestID)
	assert.Equal(t, "Finished", got[0].ContestName)
	assert.Equal(t, 2, got[0].ProblemsSolved)
	assert.Equal(t, 100, got[0].Penalty, "30 + 50 minutes plus one wrong attempt at the 20 minute default")
	assert.Equal(t, 1, got[0].Position)
}

func TestDashboardProvider_GetFinishedContestResults_NoSolvesStillListsTheContest(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	now := time.Now().UTC()
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: itContest1, GroupID: itGroupA, OwnerID: itOwner, Name: "Finished", Start: now.Add(-5 * time.Hour), End: now.Add(-3 * time.Hour)})
	itRegister(t, pool, itContest1, itOwner)

	got, err := NewDashboardProvider(pool).GetFinishedContestResults(context.Background(), itOwner, 5)

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, 0, got[0].ProblemsSolved)
	assert.Equal(t, 0, got[0].Penalty)
}

func TestDashboardProvider_GetFinishedContestResults_PositionCountsOnlyParticipantsWithMoreProblemsSolved(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	pgtest.SeedUser(t, pool, itUserC, "carla")
	now := time.Now().UTC()
	start := now.Add(-5 * time.Hour)
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: itContest1, GroupID: itGroupA, OwnerID: itOwner, Name: "Finished", Start: start, End: now.Add(-3 * time.Hour)})
	for _, u := range []string{itOwner, itUserB, itUserC} {
		itRegister(t, pool, itContest1, u)
	}
	ac := func(id, problem, user string, minute int) {
		pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: id, ProblemID: problem, UserID: user, ContestID: itContest1, Status: "ACCEPTED", SubmittedAt: start.Add(time.Duration(minute) * time.Minute)})
	}
	ac("owner-1", itProblem1, itOwner, 10)
	ac("owner-2", itProblem2, itOwner, 20)
	ac("beto-1", itProblem1, itUserB, 10)
	ac("carla-1", itProblem1, itUserC, 10)

	got, err := NewDashboardProvider(pool).GetFinishedContestResults(context.Background(), itUserB, 5)

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, 1, got[0].ProblemsSolved)
	assert.Equal(t, 2, got[0].Position, "only the owner solved more problems than beto")
}
