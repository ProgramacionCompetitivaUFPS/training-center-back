//go:build integration

package contest

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/testutil/pgtest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	itOwner    = "11111111-1111-1111-1111-111111111111"
	itUserB    = "22222222-2222-2222-2222-222222222222"
	itGroupA   = "bbbbbbbb-0000-0000-0000-00000000000a"
	itGroupB   = "bbbbbbbb-0000-0000-0000-00000000000b"
	itContest1 = "cccccccc-0000-0000-0000-000000000001"
	itContest2 = "cccccccc-0000-0000-0000-000000000002"
	itContest3 = "cccccccc-0000-0000-0000-000000000003"
	itProblem1 = "aaaaaaaa-0000-0000-0000-000000000001"
	itProblem2 = "aaaaaaaa-0000-0000-0000-000000000002"
	itMissing  = "99999999-9999-9999-9999-999999999999"
)

func itSeedBase(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	pgtest.SeedUser(t, pool, itOwner, "owner")
	pgtest.SeedUser(t, pool, itUserB, "beto")
	pgtest.SeedGroup(t, pool, itGroupA, "Group A", itOwner)
	pgtest.SeedGroup(t, pool, itGroupB, "Group B", itOwner)
	pgtest.SeedProblem(t, pool, itProblem1, "two-sum", "PUBLISHED")
	pgtest.SeedProblem(t, pool, itProblem2, "three-sum", "PUBLISHED")
}

func itContest(t *testing.T, id, name, groupID string, start, end time.Time) *domainContest.Contest {
	t.Helper()
	desc := "About " + name
	return domainContest.RestoreContest(
		id, domainContest.RestoreContestName(name), &desc,
		start.UTC(), end.UTC(),
		domainContest.RestorePenalty(20), 30, true, false, true, true,
		nil, nil,
		shared.RestoreGroupID(groupID), shared.RestoreUserID(itOwner),
		domainContest.ParticipationModeMixed, domainContest.RestoreTeamSize(2, 4),
		nil, start.UTC(), nil,
	)
}

func itUpcoming(t *testing.T, id, name, groupID string) *domainContest.Contest {
	t.Helper()
	start := time.Now().Add(24 * time.Hour).Truncate(time.Microsecond)
	return itContest(t, id, name, groupID, start, start.Add(3*time.Hour))
}

func itCreate(t *testing.T, pool *pgxpool.Pool, c *domainContest.Contest) {
	t.Helper()
	require.NoError(t, NewRepository(pool).Create(context.Background(), c))
}

func itRequireCode(t *testing.T, err error, code string) {
	t.Helper()
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, code, appErr.Code)
}

func itNames(contests []*domainContest.Contest) []string {
	out := make([]string, len(contests))
	for i, c := range contests {
		out[i] = c.Name().Value()
	}
	return out
}

func TestRepository_Create_RoundTripsEveryFieldAndProblems(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	c := itUpcoming(t, itContest1, "Weekly Contest", itGroupA)
	require.NoError(t, c.AddProblem("dddddddd-0000-0000-0000-000000000001", itProblem1, time.Now()))
	require.NoError(t, c.AddProblem("dddddddd-0000-0000-0000-000000000002", itProblem2, time.Now()))
	itCreate(t, pool, c)

	got, err := NewRepository(pool).FindByID(context.Background(), itContest1)

	require.NoError(t, err)
	assert.Equal(t, "Weekly Contest", got.Name().Value())
	require.NotNil(t, got.Description())
	assert.Equal(t, "About Weekly Contest", *got.Description())
	assert.True(t, c.StartTime().Equal(got.StartTime()))
	assert.True(t, c.EndTime().Equal(got.EndTime()))
	assert.Equal(t, 20, got.Penalty().Value())
	assert.Equal(t, 30, got.FreezeMinutes())
	assert.True(t, got.EnablePostContest())
	assert.False(t, got.Locked())
	assert.True(t, got.ShowTeamMembers())
	assert.True(t, got.AllowLateRegistration())
	assert.Equal(t, itGroupA, got.GroupID().Value())
	assert.Equal(t, itOwner, got.OwnerID().Value())
	assert.Equal(t, "MIXED", got.ParticipationMode().String())
	assert.Equal(t, 2, got.TeamSize().Min())
	assert.Equal(t, 4, got.TeamSize().Max())
	require.Len(t, got.Problems(), 2)
	assert.Equal(t, itProblem1, got.Problems()[0].ProblemID())
	assert.Equal(t, 1, got.Problems()[0].Order())
	assert.Equal(t, itProblem2, got.Problems()[1].ProblemID())
	assert.Equal(t, 2, got.Problems()[1].Order())
}

func TestRepository_Create_UnknownGroupReturnsInternalNotRawError(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)

	err := NewRepository(pool).Create(context.Background(), itUpcoming(t, itContest1, "Orphan", itMissing))

	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.KindInternal, appErr.Kind)
}

func TestRepository_FindByID_MissingAndMalformedReturnNotFound(t *testing.T) {
	pool := pgtest.NewDB(t)
	repo := NewRepository(pool)

	_, err := repo.FindByID(context.Background(), itMissing)
	itRequireCode(t, err, domainContest.ErrCodeContestNotFound)

	_, err = repo.FindByID(context.Background(), "not-a-uuid")
	itRequireCode(t, err, domainContest.ErrCodeContestNotFound)
}

func TestRepository_Update_PersistsChangesAndReplacesProblems(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	c := itUpcoming(t, itContest1, "Weekly Contest", itGroupA)
	require.NoError(t, c.AddProblem("dddddddd-0000-0000-0000-000000000001", itProblem1, time.Now()))
	itCreate(t, pool, c)
	repo := NewRepository(pool)

	loaded, err := repo.FindByID(context.Background(), itContest1)
	require.NoError(t, err)
	now := time.Now()
	loaded.SetLocked(true, now)
	loaded.SetShowTeamMembers(false, now)
	loaded.SetProblems([]domainContest.ContestProblem{
		domainContest.RestoreContestProblem("dddddddd-0000-0000-0000-000000000009", itProblem2, 1, nil),
	}, now)
	require.NoError(t, repo.Update(context.Background(), loaded))

	got, err := repo.FindByID(context.Background(), itContest1)
	require.NoError(t, err)
	assert.True(t, got.Locked())
	assert.False(t, got.ShowTeamMembers())
	require.Len(t, got.Problems(), 1)
	assert.Equal(t, itProblem2, got.Problems()[0].ProblemID())
}

func TestRepository_Update_StaleCopyReturnsConflict(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	itCreate(t, pool, itUpcoming(t, itContest1, "Weekly Contest", itGroupA))
	repo := NewRepository(pool)
	first, err := repo.FindByID(context.Background(), itContest1)
	require.NoError(t, err)
	second, err := repo.FindByID(context.Background(), itContest1)
	require.NoError(t, err)
	first.SetLocked(true, time.Now())
	require.NoError(t, repo.Update(context.Background(), first))

	second.SetShowTeamMembers(false, time.Now())
	err = repo.Update(context.Background(), second)

	itRequireCode(t, err, domainContest.ErrCodeContestConflict)
}

func TestRepository_Update_MissingReturnsNotFound(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)

	err := NewRepository(pool).Update(context.Background(), itUpcoming(t, itContest1, "Ghost", itGroupA))

	itRequireCode(t, err, domainContest.ErrCodeContestNotFound)
}

func TestRepository_Delete_RemovesAnUnlockedContestAndItsProblems(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	c := itUpcoming(t, itContest1, "Weekly Contest", itGroupA)
	require.NoError(t, c.AddProblem("dddddddd-0000-0000-0000-000000000001", itProblem1, time.Now()))
	itCreate(t, pool, c)
	repo := NewRepository(pool)

	require.NoError(t, repo.Delete(context.Background(), itContest1))

	_, err := repo.FindByID(context.Background(), itContest1)
	itRequireCode(t, err, domainContest.ErrCodeContestNotFound)
	var remaining int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM contest_problems`).Scan(&remaining))
	assert.Equal(t, 0, remaining)
}

func TestRepository_Delete_LockedContestIsForbiddenAndMissingIsNotFound(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	c := itUpcoming(t, itContest1, "Weekly Contest", itGroupA)
	itCreate(t, pool, c)
	repo := NewRepository(pool)
	loaded, err := repo.FindByID(context.Background(), itContest1)
	require.NoError(t, err)
	loaded.SetLocked(true, time.Now())
	require.NoError(t, repo.Update(context.Background(), loaded))

	itRequireCode(t, repo.Delete(context.Background(), itContest1), domainContest.ErrCodeContestLocked)
	itRequireCode(t, repo.Delete(context.Background(), itMissing), domainContest.ErrCodeContestNotFound)

	stillThere, err := repo.FindByID(context.Background(), itContest1)
	require.NoError(t, err)
	assert.NotNil(t, stillThere)
}

func seedCatalog(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	now := time.Now()
	itCreate(t, pool, itContest(t, itContest1, "Alpha Past", itGroupA, now.Add(-48*time.Hour), now.Add(-24*time.Hour)))
	itCreate(t, pool, itContest(t, itContest2, "Beta Running", itGroupA, now.Add(-time.Hour), now.Add(time.Hour)))
	itCreate(t, pool, itContest(t, itContest3, "Gamma 100% Future", itGroupB, now.Add(24*time.Hour), now.Add(48*time.Hour)))
}

func TestRepository_List_ScopesToTheGroupAndFilters(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	seedCatalog(t, pool)
	repo := NewRepository(pool)
	finished, active, scheduled := domainContest.StatusFinished, domainContest.StatusActive, domainContest.StatusScheduled

	tests := []struct {
		name      string
		filters   domainContest.ListFilters
		wantNames []string
		wantTotal int
	}{
		{"group A newest start first", domainContest.ListFilters{GroupID: shared.RestoreGroupID(itGroupA)}, []string{"Beta Running", "Alpha Past"}, 2},
		{"group B only", domainContest.ListFilters{GroupID: shared.RestoreGroupID(itGroupB)}, []string{"Gamma 100% Future"}, 1},
		{"ascending by name", domainContest.ListFilters{GroupID: shared.RestoreGroupID(itGroupA), SortBy: domainContest.SortByName, Order: domainContest.OrderAsc}, []string{"Alpha Past", "Beta Running"}, 2},
		{"finished", domainContest.ListFilters{GroupID: shared.RestoreGroupID(itGroupA), Status: &finished}, []string{"Alpha Past"}, 1},
		{"active", domainContest.ListFilters{GroupID: shared.RestoreGroupID(itGroupA), Status: &active}, []string{"Beta Running"}, 1},
		{"scheduled", domainContest.ListFilters{GroupID: shared.RestoreGroupID(itGroupB), Status: &scheduled}, []string{"Gamma 100% Future"}, 1},
		{"search case insensitive", domainContest.ListFilters{GroupID: shared.RestoreGroupID(itGroupA), Search: "BETA"}, []string{"Beta Running"}, 1},
		{"percent in search is literal", domainContest.ListFilters{GroupID: shared.RestoreGroupID(itGroupB), Search: "%"}, []string{"Gamma 100% Future"}, 1},
		{"page two of size one", domainContest.ListFilters{GroupID: shared.RestoreGroupID(itGroupA), Page: 2, Limit: 1}, []string{"Alpha Past"}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.filters.Page == 0 {
				tt.filters.Page = 1
			}
			if tt.filters.Limit == 0 {
				tt.filters.Limit = 10
			}

			got, total, err := repo.List(context.Background(), tt.filters)

			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, total)
			assert.Equal(t, tt.wantNames, itNames(got))
		})
	}
}

func TestRepository_ListByGroupIDs_NilMeansAllAndEmptyMeansNone(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	seedCatalog(t, pool)
	repo := NewRepository(pool)
	filters := domainContest.ListFilters{Page: 1, Limit: 10, SortBy: domainContest.SortByName, Order: domainContest.OrderAsc}

	all, total, err := repo.ListByGroupIDs(context.Background(), nil, filters)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Equal(t, []string{"Alpha Past", "Beta Running", "Gamma 100% Future"}, itNames(all))

	none, total, err := repo.ListByGroupIDs(context.Background(), []string{}, filters)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, none)

	onlyB, total, err := repo.ListByGroupIDs(context.Background(), []string{itGroupB}, filters)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, []string{"Gamma 100% Future"}, itNames(onlyB))
}

func TestRepository_List_LoadsTheProblemsOfEachContestInOrder(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: itContest1, GroupID: itGroupA, OwnerID: itOwner, Name: "One", Start: time.Now().Add(time.Hour), End: time.Now().Add(2 * time.Hour)})
	pgtest.SeedContestProblem(t, pool, "dddddddd-0000-0000-0000-000000000002", itContest1, itProblem2, 2)
	pgtest.SeedContestProblem(t, pool, "dddddddd-0000-0000-0000-000000000001", itContest1, itProblem1, 1)

	got, _, err := NewRepository(pool).List(context.Background(), domainContest.ListFilters{GroupID: shared.RestoreGroupID(itGroupA), Page: 1, Limit: 10})

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Len(t, got[0].Problems(), 2)
	assert.Equal(t, itProblem1, got[0].Problems()[0].ProblemID())
	assert.Equal(t, itProblem2, got[0].Problems()[1].ProblemID())
}
