//go:build integration

package submission

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/internal/testutil/pgtest"
)

const itMissing = "99999999-9999-9999-9999-999999999999"

func TestUserProvider_GetUserByID_FoundAndMissing(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	provider := NewUserProvider(pool)

	got, err := provider.GetUserByID(context.Background(), itUserA)
	require.NoError(t, err)
	assert.Equal(t, itUserA, got.ID)
	assert.Equal(t, "ana", got.Nickname)

	_, err = provider.GetUserByID(context.Background(), itMissing)
	assertAppErrorCode(t, err, user.ErrCodeUserNotFound)

	_, err = provider.GetUserByID(context.Background(), "not-a-uuid")
	assertAppErrorCode(t, err, user.ErrCodeUserNotFound)
}

func TestUserProvider_GetUsersByIDs_ReturnsOnlyExisting(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	provider := NewUserProvider(pool)

	got, err := provider.GetUsersByIDs(context.Background(), []string{itUserA, itUserB, itMissing})
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "beto", got[itUserB].Nickname)

	empty, err := provider.GetUsersByIDs(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestProblemProvider_GetProblemBySlug_MapsStatusAccessModifiersAndTests(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	_, err := pool.Exec(context.Background(),
		`UPDATE problems SET modifiers_ids = ARRAY[$1, $2]::uuid[], test_cases_key = 'tests.zip' WHERE id = $3`,
		itUserA, itUserB, itProblem1)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), `UPDATE problems SET status = 'DRAFT', accessibility = 'PRIVATE' WHERE id = $1`, itProblem2)
	require.NoError(t, err)
	provider := NewProblemProvider(pool)

	published, err := provider.GetProblemBySlug(context.Background(), "two-sum")
	require.NoError(t, err)
	assert.Equal(t, itProblem1, published.ID)
	assert.Equal(t, pgtest.SystemUserID, published.AuthorID)
	assert.True(t, published.IsPublished)
	assert.True(t, published.IsPublic)
	assert.True(t, published.HasTestCases)
	assert.Equal(t, []string{itUserA, itUserB}, published.ModifierIDs)

	draft, err := provider.GetProblemBySlug(context.Background(), "three-sum")
	require.NoError(t, err)
	assert.False(t, draft.IsPublished)
	assert.False(t, draft.IsPublic)
	assert.False(t, draft.HasTestCases)
	assert.Empty(t, draft.ModifierIDs)

	_, err = provider.GetProblemBySlug(context.Background(), "nope")
	assertAppErrorCode(t, err, problem.ErrCodeProblemNotFound)
}

func TestProblemDisplayProvider_GetProblemByIDAndGetProblemsByIDs(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	provider := NewProblemDisplayProvider(pool)

	one, err := provider.GetProblemByID(context.Background(), itProblem1)
	require.NoError(t, err)
	assert.Equal(t, "two-sum", one.Slug)
	assert.Equal(t, "two-sum", one.Title)

	_, err = provider.GetProblemByID(context.Background(), itMissing)
	assertAppErrorCode(t, err, problem.ErrCodeProblemNotFound)

	many, err := provider.GetProblemsByIDs(context.Background(), []string{itProblem1, itProblem2, itMissing})
	require.NoError(t, err)
	require.Len(t, many, 2)
	assert.Equal(t, "three-sum", many[itProblem2].Slug)

	empty, err := provider.GetProblemsByIDs(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestProblemJudgingProvider_GetJudgingUpdatedAtAndIsPublished(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	updated := itNow.Add(time.Hour)
	_, err := pool.Exec(context.Background(), `UPDATE problems SET judging_updated_at = $1 WHERE id = $2`, updated, itProblem1)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), `UPDATE problems SET status = 'DRAFT' WHERE id = $1`, itProblem2)
	require.NoError(t, err)
	provider := NewProblemJudgingProvider(pool)

	at, err := provider.GetJudgingUpdatedAt(context.Background(), itProblem1)
	require.NoError(t, err)
	require.NotNil(t, at)
	assert.True(t, updated.Equal(*at))

	never, err := provider.GetJudgingUpdatedAt(context.Background(), itProblem2)
	require.NoError(t, err)
	assert.Nil(t, never)

	missing, err := provider.GetJudgingUpdatedAt(context.Background(), itMissing)
	require.NoError(t, err)
	assert.Nil(t, missing)

	published, err := provider.IsPublished(context.Background(), itProblem1)
	require.NoError(t, err)
	assert.True(t, published)

	draft, err := provider.IsPublished(context.Background(), itProblem2)
	require.NoError(t, err)
	assert.False(t, draft)

	absent, err := provider.IsPublished(context.Background(), itMissing)
	require.NoError(t, err)
	assert.False(t, absent)

	empty, err := provider.IsPublished(context.Background(), "")
	require.NoError(t, err)
	assert.False(t, empty)
}

func TestDashboardProvider_GetRecentSubmissions_NewestFirstLimitedToTheUser(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "old", ProblemID: itProblem1, UserID: itUserA, Status: "WRONG_ANSWER", SubmittedAt: itNow})
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "new", ProblemID: itProblem1, UserID: itUserA, Status: "ACCEPTED", Language: "java17", SubmittedAt: itNow.Add(time.Hour)})
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "middle", ProblemID: itProblem2, UserID: itUserA, SubmittedAt: itNow.Add(30 * time.Minute)})
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "theirs", ProblemID: itProblem1, UserID: itUserB, SubmittedAt: itNow.Add(2 * time.Hour)})
	_, err := pool.Exec(context.Background(),
		`UPDATE submissions SET problem_slug = 'two-sum', problem_title = 'Two Sum', time_ms = 120, memory_kb = 2048 WHERE id = 'new'`)
	require.NoError(t, err)
	provider := NewDashboardProvider(pool)

	got, err := provider.GetRecentSubmissions(context.Background(), itUserA, 2)

	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "new", got[0].ID)
	assert.Equal(t, "two-sum", got[0].ProblemSlug)
	assert.Equal(t, "Two Sum", got[0].ProblemTitle)
	assert.Equal(t, "ACCEPTED", got[0].Verdict)
	assert.Equal(t, "java17", got[0].Language)
	require.NotNil(t, got[0].ExecutionTime)
	assert.Equal(t, 120, *got[0].ExecutionTime)
	require.NotNil(t, got[0].MemoryKb)
	assert.Equal(t, 2048, *got[0].MemoryKb)
	assert.Equal(t, "middle", got[1].ID)
	assert.Equal(t, "", got[1].ProblemSlug)
	assert.Nil(t, got[1].ExecutionTime)
}

func TestDashboardProvider_GetSubmissionDates_DistinctUTCDaysAscending(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	day1 := time.Date(2026, 8, 1, 23, 30, 0, 0, time.UTC)
	day2 := time.Date(2026, 8, 3, 0, 15, 0, 0, time.UTC)
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "d2a", ProblemID: itProblem1, UserID: itUserA, SubmittedAt: day2})
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "d1a", ProblemID: itProblem1, UserID: itUserA, SubmittedAt: day1})
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "d1b", ProblemID: itProblem2, UserID: itUserA, SubmittedAt: day1.Add(-time.Hour)})
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "other", ProblemID: itProblem1, UserID: itUserB, SubmittedAt: day2.Add(24 * time.Hour)})

	got, err := NewDashboardProvider(pool).GetSubmissionDates(context.Background(), itUserA)

	require.NoError(t, err)
	assert.Equal(t, []time.Time{
		time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC),
	}, got)
}

func TestSubmissionStatsProvider_GetSubmissionCounts(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "s1", ProblemID: itProblem1, UserID: itUserA, Status: "ACCEPTED"})
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "s2", ProblemID: itProblem1, UserID: itUserA, Status: "WRONG_ANSWER"})
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "s3", ProblemID: itProblem2, UserID: itUserA, Status: "ACCEPTED"})
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "s4", ProblemID: itProblem1, UserID: itUserB, Status: "ACCEPTED"})
	provider := NewSubmissionStatsProvider(pool)

	total, accepted, err := provider.GetSubmissionCounts(context.Background(), itUserA)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Equal(t, 2, accepted)

	total, accepted, err = provider.GetSubmissionCounts(context.Background(), itMissing)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Equal(t, 0, accepted)
}
