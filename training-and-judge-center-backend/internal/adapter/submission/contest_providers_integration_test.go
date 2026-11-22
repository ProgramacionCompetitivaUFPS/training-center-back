//go:build integration

package submission

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/testutil/pgtest"
)

const (
	itGroupA     = "bbbbbbbb-0000-0000-0000-00000000000a"
	itGroupB     = "bbbbbbbb-0000-0000-0000-00000000000b"
	itDefaultGrp = "00000000-0000-0000-0000-000000000002"
	itContest1   = "cccccccc-0000-0000-0000-000000000001"
	itContest2   = "cccccccc-0000-0000-0000-000000000002"
	itTeam1      = "eeeeeeee-0000-0000-0000-000000000001"
	itUserC      = "33333333-3333-3333-3333-333333333333"
)

func itSeedContests(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	itSeedBase(t, pool)
	pgtest.SeedUser(t, pool, itUserC, "carla")
	pgtest.SeedGroup(t, pool, itGroupA, "Group A", itUserA)
	pgtest.SeedGroup(t, pool, itGroupB, "Group B", itUserA)
	start := time.Now().Add(time.Hour).Truncate(time.Microsecond)
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: itContest1, GroupID: itGroupA, OwnerID: itUserA, Name: "Contest One", Start: start, End: start.Add(2 * time.Hour)})
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: itContest2, GroupID: itGroupB, OwnerID: itUserA, Name: "Contest Two", Start: start, End: start.Add(time.Hour)})
	pgtest.SeedContestProblem(t, pool, "dddddddd-0000-0000-0000-000000000001", itContest1, itProblem1, 1)
	pgtest.SeedContestProblem(t, pool, "dddddddd-0000-0000-0000-000000000002", itContest1, itProblem2, 2)
}

func TestContestProvider_GetContestByIDAndGetContestsByIDs(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedContests(t, pool)
	provider := NewContestProvider(pool)

	got, err := provider.GetContestByID(context.Background(), itContest1)
	require.NoError(t, err)
	assert.Equal(t, "Contest One", got.Name)
	assert.Equal(t, itGroupA, got.GroupID)

	_, err = provider.GetContestByID(context.Background(), itMissing)
	assertAppErrorCode(t, err, domainContest.ErrCodeContestNotFound)

	_, err = provider.GetContestByID(context.Background(), "not-a-uuid")
	assertAppErrorCode(t, err, domainContest.ErrCodeContestNotFound)

	many, err := provider.GetContestsByIDs(context.Background(), []string{itContest1, itContest2, itMissing})
	require.NoError(t, err)
	require.Len(t, many, 2)
	assert.Equal(t, "Contest Two", many[itContest2].Name)

	empty, err := provider.GetContestsByIDs(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestContestProvider_GetProblemPosition(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedContests(t, pool)
	provider := NewContestProvider(pool)

	second, err := provider.GetProblemPosition(context.Background(), itContest1, itProblem2)
	require.NoError(t, err)
	require.NotNil(t, second)
	assert.Equal(t, 2, *second)

	notIn, err := provider.GetProblemPosition(context.Background(), itContest2, itProblem1)
	require.NoError(t, err)
	assert.Nil(t, notIn)
}

func TestContestSubmissionProvider_GetContestForSubmission_RequiresTheMatchingGroup(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedContests(t, pool)
	provider := NewContestSubmissionProvider(pool)

	got, err := provider.GetContestForSubmission(context.Background(), itGroupA, itContest1)
	require.NoError(t, err)
	assert.Equal(t, "Contest One", got.Name)
	assert.False(t, got.EnablePostContest)
	assert.True(t, got.EndTime.After(got.StartTime))
	assert.ElementsMatch(t, []string{itProblem1, itProblem2}, got.ProblemIDs)

	_, err = provider.GetContestForSubmission(context.Background(), itGroupB, itContest1)
	assertAppErrorCode(t, err, domainContest.ErrCodeContestNotFound)
}

func TestContestTimesProvider_GetContestTimes(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedContests(t, pool)
	provider := NewContestTimesProvider(pool)

	start, end, err := provider.GetContestTimes(context.Background(), itContest2)
	require.NoError(t, err)
	assert.Equal(t, time.Hour, end.Sub(start))

	_, _, err = provider.GetContestTimes(context.Background(), itMissing)
	assertAppErrorCode(t, err, domainContest.ErrCodeContestNotFound)
}

func TestGroupMembershipChecker_IsGroupMember(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedContests(t, pool)
	pgtest.SeedGroupMember(t, pool, "77770000-0000-0000-0000-0000000000b1", itGroupA, itUserB, "MEMBER")
	checker := NewGroupMembershipChecker(pool)

	member, err := checker.IsGroupMember(context.Background(), itGroupA, itUserB)
	require.NoError(t, err)
	assert.True(t, member)

	otherGroup, err := checker.IsGroupMember(context.Background(), itGroupB, itUserB)
	require.NoError(t, err)
	assert.False(t, otherGroup)
}

func TestLeadChecker_ContestGroupLeadAndLeadOfMember(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedContests(t, pool)
	pgtest.SeedGroupMember(t, pool, "77770000-0000-0000-0000-0000000000b1", itGroupA, itUserA, "LEAD")
	pgtest.SeedGroupMember(t, pool, "77770000-0000-0000-0000-0000000000b2", itGroupA, itUserB, "MEMBER")
	pgtest.SeedGroupMember(t, pool, "77770000-0000-0000-0000-0000000000b3", itDefaultGrp, itUserA, "LEAD")
	pgtest.SeedGroupMember(t, pool, "77770000-0000-0000-0000-0000000000b4", itDefaultGrp, itUserC, "MEMBER")
	checker := NewLeadChecker(pool)

	lead, err := checker.IsLeadOfContestGroup(context.Background(), itContest1, itUserA)
	require.NoError(t, err)
	assert.True(t, lead)
	member, err := checker.IsLeadOfContestGroup(context.Background(), itContest1, itUserB)
	require.NoError(t, err)
	assert.False(t, member)
	wrongContest, err := checker.IsLeadOfContestGroup(context.Background(), itContest2, itUserA)
	require.NoError(t, err)
	assert.False(t, wrongContest)

	leadsMember, err := checker.IsLeadOfMember(context.Background(), itUserA, itUserB)
	require.NoError(t, err)
	assert.True(t, leadsMember)
	notLead, err := checker.IsLeadOfMember(context.Background(), itUserB, itUserA)
	require.NoError(t, err)
	assert.False(t, notLead)
	onlyDefaultGroup, err := checker.IsLeadOfMember(context.Background(), itUserA, itUserC)
	require.NoError(t, err)
	assert.False(t, onlyDefaultGroup, "the default group does not make someone a lead of its members")
}

func TestTeamMembershipChecker_IsTeammate(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedContests(t, pool)
	pgtest.SeedTeam(t, pool, itTeam1, "Team One", itUserA)
	pgtest.SeedTeamMember(t, pool, "66660000-0000-0000-0000-000000000001", itTeam1, itUserA)
	pgtest.SeedTeamMember(t, pool, "66660000-0000-0000-0000-000000000002", itTeam1, itUserB)
	pgtest.SeedTeamMember(t, pool, "66660000-0000-0000-0000-000000000003", itTeam1, itUserC)
	pgtest.SeedTeamParticipation(t, pool, "99990000-0000-0000-0000-000000000001", itContest1, itTeam1, itUserA, itUserB)
	checker := NewTeamMembershipChecker(pool)

	tests := []struct {
		name           string
		viewer, author string
		want           bool
	}{
		{"both selected in the same team", itUserA, itUserB, true},
		{"the author was not selected", itUserA, itUserC, false},
		{"the viewer was not selected", itUserC, itUserA, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checker.IsTeammate(context.Background(), itContest1, tt.viewer, tt.author)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStandingIDResolver_IndividualThenTeamThenNothing(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedContests(t, pool)
	pgtest.SeedTeam(t, pool, itTeam1, "Team One", itUserA)
	pgtest.SeedTeamMember(t, pool, "66660000-0000-0000-0000-000000000002", itTeam1, itUserB)
	pgtest.SeedTeamParticipation(t, pool, "99990000-0000-0000-0000-000000000001", itContest1, itTeam1, itUserB)
	pgtest.SeedRegistration(t, pool, "ffffffff-0000-0000-0000-000000000001", itContest2, itUserC)
	resolver := NewStandingIDResolver(pool)

	standing, ok, err := resolver.ResolveStandingID(context.Background(), itContest2, itUserC)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, itUserC, standing)

	standing, ok, err = resolver.ResolveStandingID(context.Background(), itContest1, itUserB)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, itTeam1, standing)

	standing, ok, err = resolver.ResolveStandingID(context.Background(), itContest1, itUserC)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, "", standing)
}
