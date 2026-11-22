//go:build integration

package problem

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/training-judge-center/backend/internal/testutil/pgtest"
)

const (
	pGroup    = "bbbbbbbb-0000-0000-0000-00000000000a"
	pContest1 = "cccccccc-0000-0000-0000-000000000001"
	pContest2 = "cccccccc-0000-0000-0000-000000000002"
	pMissing  = "99999999-9999-9999-9999-999999999999"
)

func pSeedBase(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	pgtest.SeedUser(t, pool, authorA, "ana")
	pgtest.SeedUser(t, pool, authorB, "beto")
	pgtest.SeedProblem(t, pool, idOne, "two-sum", "PUBLISHED")
	pgtest.SeedProblem(t, pool, idTwo, "three-sum", "DRAFT")
	pgtest.SeedGroup(t, pool, pGroup, "Group A", pgtest.SystemUserID)
}

func pSubmit(t *testing.T, pool *pgxpool.Pool, id, problemID, user, status, language, contestID string) {
	t.Helper()
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: id, ProblemID: problemID, UserID: user, Status: status, Language: language, ContestID: contestID})
}

func TestStatisticsProvider_GetByProblemID_AggregatesTotalsLanguagesAndVerdicts(t *testing.T) {
	pool := pgtest.NewDB(t)
	pSeedBase(t, pool)
	pSubmit(t, pool, "s1", idOne, authorA, "ACCEPTED", "cpp20", "")
	pSubmit(t, pool, "s2", idOne, authorA, "ACCEPTED", "cpp20", "")
	pSubmit(t, pool, "s3", idOne, authorB, "WRONG_ANSWER", "cpp20", "")
	pSubmit(t, pool, "s4", idOne, authorB, "ACCEPTED", "python310", "")
	pSubmit(t, pool, "other-problem", idTwo, authorA, "ACCEPTED", "java17", "")

	got, err := NewStatisticsProvider(pool).GetByProblemID(context.Background(), idOne)

	require.NoError(t, err)
	assert.Equal(t, 4, got.TotalSubmissions)
	assert.Equal(t, 2, got.UniqueAttempted)
	assert.Equal(t, 2, got.UniqueSolved)
	require.Len(t, got.ByLanguage, 2)
	assert.Equal(t, "python310", got.ByLanguage[0].Language, "ordered by acceptance rate, best first")
	assert.Equal(t, 1, got.ByLanguage[0].UsersAccepted)
	assert.Equal(t, "cpp20", got.ByLanguage[1].Language)
	assert.Equal(t, 2, got.ByLanguage[1].UsersAttempted)
	assert.Equal(t, 1, got.ByLanguage[1].UsersAccepted)
	require.Len(t, got.ByVerdict, 2)
	assert.Equal(t, "ACCEPTED", got.ByVerdict[0].Verdict)
	assert.Equal(t, 3, got.ByVerdict[0].Count)
	assert.Equal(t, "WRONG_ANSWER", got.ByVerdict[1].Verdict)
}

func TestStatisticsProvider_GetByProblemID_NoSubmissionsOrMalformedIDGivesZeroTotals(t *testing.T) {
	pool := pgtest.NewDB(t)
	pSeedBase(t, pool)
	provider := NewStatisticsProvider(pool)

	empty, err := provider.GetByProblemID(context.Background(), idTwo)
	require.NoError(t, err)
	assert.Equal(t, 0, empty.TotalSubmissions)
	assert.Empty(t, empty.ByLanguage)
	assert.Empty(t, empty.ByVerdict)

	malformed, err := provider.GetByProblemID(context.Background(), "not-a-uuid")
	require.NoError(t, err)
	assert.Equal(t, 0, malformed.TotalSubmissions)
}

func TestTopicStatsProvider_GetTopicBreakdown_CountsDistinctSolvedProblemsPerTag(t *testing.T) {
	pool := pgtest.NewDB(t)
	pSeedBase(t, pool)
	_, err := pool.Exec(context.Background(), `UPDATE problems SET tags = ARRAY['graphs','dp'] WHERE id = $1`, idOne)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), `UPDATE problems SET tags = ARRAY['dp'] WHERE id = $1`, idTwo)
	require.NoError(t, err)
	pSubmit(t, pool, "s1", idOne, authorA, "ACCEPTED", "cpp20", "")
	pSubmit(t, pool, "s2", idOne, authorA, "ACCEPTED", "cpp20", "")
	pSubmit(t, pool, "s3", idTwo, authorA, "ACCEPTED", "cpp20", "")
	pSubmit(t, pool, "s4", idTwo, authorB, "WRONG_ANSWER", "cpp20", "")
	provider := NewTopicStatsProvider(pool)

	got, err := provider.GetTopicBreakdown(context.Background(), authorA)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "dp", got[0].Tag)
	assert.Equal(t, 2, got[0].Solved)
	assert.Equal(t, "graphs", got[1].Tag)
	assert.Equal(t, 1, got[1].Solved)

	none, err := provider.GetTopicBreakdown(context.Background(), authorB)
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestActiveContestChecker_IsProblemInActiveContest_OnlyWhileTheContestIsRunning(t *testing.T) {
	pool := pgtest.NewDB(t)
	pSeedBase(t, pool)
	now := time.Now()
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: pContest1, GroupID: pGroup, OwnerID: authorA, Start: now.Add(-time.Hour), End: now.Add(time.Hour)})
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: pContest2, GroupID: pGroup, OwnerID: authorA, Start: now.Add(24 * time.Hour), End: now.Add(26 * time.Hour)})
	pgtest.SeedContestProblem(t, pool, "dddddddd-0000-0000-0000-000000000001", pContest1, idOne, 1)
	pgtest.SeedContestProblem(t, pool, "dddddddd-0000-0000-0000-000000000002", pContest2, idTwo, 1)
	checker := NewActiveContestChecker(pool)

	running, err := checker.IsProblemInActiveContest(context.Background(), idOne)
	require.NoError(t, err)
	assert.True(t, running)

	scheduled, err := checker.IsProblemInActiveContest(context.Background(), idTwo)
	require.NoError(t, err)
	assert.False(t, scheduled)
}

func TestActiveContestChecker_HasContestSubmissions(t *testing.T) {
	pool := pgtest.NewDB(t)
	pSeedBase(t, pool)
	now := time.Now()
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: pContest1, GroupID: pGroup, OwnerID: authorA, Start: now.Add(-time.Hour), End: now.Add(time.Hour)})
	pSubmit(t, pool, "in-contest", idOne, authorA, "ACCEPTED", "cpp20", pContest1)
	pSubmit(t, pool, "practice", idTwo, authorA, "ACCEPTED", "cpp20", "")
	checker := NewActiveContestChecker(pool)

	withContest, err := checker.HasContestSubmissions(context.Background(), idOne)
	require.NoError(t, err)
	assert.True(t, withContest)

	practiceOnly, err := checker.HasContestSubmissions(context.Background(), idTwo)
	require.NoError(t, err)
	assert.False(t, practiceOnly)
}

func TestAffectedContestsProvider_ListOpenContestsOfProblem_ExcludesFinishedOrderedByStart(t *testing.T) {
	pool := pgtest.NewDB(t)
	pSeedBase(t, pool)
	now := time.Now()
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: pContest1, GroupID: pGroup, OwnerID: authorA, Name: "Later", Start: now.Add(24 * time.Hour), End: now.Add(26 * time.Hour)})
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: pContest2, GroupID: pGroup, OwnerID: authorA, Name: "Running", Start: now.Add(-time.Hour), End: now.Add(time.Hour)})
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: "cccccccc-0000-0000-0000-000000000003", GroupID: pGroup, OwnerID: authorA, Name: "Finished", Start: now.Add(-48 * time.Hour), End: now.Add(-24 * time.Hour)})
	pgtest.SeedContestProblem(t, pool, "dddddddd-0000-0000-0000-000000000001", pContest1, idOne, 1)
	pgtest.SeedContestProblem(t, pool, "dddddddd-0000-0000-0000-000000000002", pContest2, idOne, 1)
	pgtest.SeedContestProblem(t, pool, "dddddddd-0000-0000-0000-000000000003", "cccccccc-0000-0000-0000-000000000003", idOne, 1)

	got, err := NewAffectedContestsProvider(pool).ListOpenContestsOfProblem(context.Background(), idOne)

	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "Running", got[0].Name)
	assert.Equal(t, "Later", got[1].Name)
	assert.Equal(t, pContest2, got[0].ID)

	none, err := NewAffectedContestsProvider(pool).ListOpenContestsOfProblem(context.Background(), idTwo)
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestProblemStatusProvider_GetStatus(t *testing.T) {
	pool := pgtest.NewDB(t)
	pSeedBase(t, pool)
	provider := NewProblemStatusProvider(pool)

	published, err := provider.GetStatus(context.Background(), idOne)
	require.NoError(t, err)
	assert.Equal(t, "PUBLISHED", published)

	draft, err := provider.GetStatus(context.Background(), idTwo)
	require.NoError(t, err)
	assert.Equal(t, "DRAFT", draft)

	_, err = provider.GetStatus(context.Background(), pMissing)
	requireProblemNotFound(t, err)

	_, err = provider.GetStatus(context.Background(), "not-a-uuid")
	requireProblemNotFound(t, err)
}
