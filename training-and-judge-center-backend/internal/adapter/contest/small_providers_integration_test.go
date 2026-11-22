//go:build integration

package contest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/testutil/pgtest"
)

func TestGroupMemberProvider_GetMemberRoleAndBulk(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedWithContests(t, pool)
	pgtest.SeedGroupMember(t, pool, "77770000-0000-0000-0000-0000000000b1", itGroupA, itOwner, "LEAD")
	pgtest.SeedGroupMember(t, pool, "77770000-0000-0000-0000-0000000000b2", itGroupB, itOwner, "MEMBER")
	provider := NewGroupMemberProvider(pool)

	lead, err := provider.GetMemberRole(context.Background(), itOwner, itGroupA)
	require.NoError(t, err)
	require.NotNil(t, lead)
	assert.Equal(t, "LEAD", *lead)

	none, err := provider.GetMemberRole(context.Background(), itUserB, itGroupA)
	require.NoError(t, err)
	assert.Nil(t, none)

	malformed, err := provider.GetMemberRole(context.Background(), itOwner, "not-a-uuid")
	require.NoError(t, err)
	assert.Nil(t, malformed)

	bulk, err := provider.GetMemberRoleBulk(context.Background(), itOwner, []string{itGroupA, itGroupB, itMissing})
	require.NoError(t, err)
	require.Len(t, bulk, 2)
	assert.Equal(t, "LEAD", *bulk[itGroupA])
	assert.Equal(t, "MEMBER", *bulk[itGroupB])

	empty, err := provider.GetMemberRoleBulk(context.Background(), itOwner, nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestOwnerProvider_GetDisplay_FoundAndMissing(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	provider := NewOwnerProvider(pool)

	got, err := provider.GetDisplay(context.Background(), itOwner)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, itOwner, got.ID)
	assert.Equal(t, "owner", got.Nickname)

	missing, err := provider.GetDisplay(context.Background(), itMissing)
	require.NoError(t, err)
	assert.Nil(t, missing)
}

func TestCallerStandingProvider_ResolvesIndividualThenTeamThenNothing(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedWithContests(t, pool)
	itSeedTeams(t, pool)
	itRegister(t, pool, itContest1, itUserB)
	itSeedTeamEntry(t, pool, itPart1, itContest1, itTeam1, itOwner)
	provider := NewCallerStandingProvider(pool)

	standing, ok, err := provider.GetCallerStandingID(context.Background(), itContest1, itUserB)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, itUserB, standing)

	standing, ok, err = provider.GetCallerStandingID(context.Background(), itContest1, itOwner)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, itTeam1, standing)

	standing, ok, err = provider.GetCallerStandingID(context.Background(), itContest1, itUserC)
	require.NoError(t, err)
	assert.False(t, ok, "a team member who was not selected has no standing")
	assert.Equal(t, "", standing)
}

func TestContestParticipantProvider_DelegatesToTheRegistrations(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedWithContests(t, pool)
	itSeedTeams(t, pool)
	itRegister(t, pool, itContest1, itUserB)
	itSeedTeamEntry(t, pool, itPart1, itContest1, itTeam1, itOwner)
	provider := NewContestParticipantProvider(NewRegistrationRepository(pool))

	registered, err := provider.IsRegistered(context.Background(), itContest1, itOwner)
	require.NoError(t, err)
	assert.True(t, registered)

	count, err := provider.CountParticipants(context.Background(), itContest1)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	counts, err := provider.CountParticipantsBulk(context.Background(), []string{itContest1, itContest2})
	require.NoError(t, err)
	assert.Equal(t, map[string]int{itContest1: 2, itContest2: 0}, counts)

	flags, err := provider.IsRegisteredBulk(context.Background(), []string{itContest1, itContest2}, itUserB)
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{itContest1: true, itContest2: false}, flags)
}

func TestContestParticipationProvider_CountsContestsAsIndividualOrSelectedTeamMember(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedWithContests(t, pool)
	itSeedTeams(t, pool)
	itRegister(t, pool, itContest1, itOwner)
	itSeedTeamEntry(t, pool, itPart1, itContest2, itTeam1, itOwner)
	provider := NewContestParticipationProvider(pool)

	both, err := provider.GetContestsParticipatedCount(context.Background(), itOwner)
	require.NoError(t, err)
	assert.Equal(t, 2, both)

	notSelected, err := provider.GetContestsParticipatedCount(context.Background(), itUserC)
	require.NoError(t, err)
	assert.Equal(t, 0, notSelected)
}

func TestParticipantProviders_NicknamesProfilesAndTeamDisplays(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedWithContests(t, pool)
	itSeedTeams(t, pool)

	nicknames, err := NewParticipantNicknameProvider(pool).GetNicknamesByIDs(context.Background(), []string{itOwner, itUserB, itMissing})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{itOwner: "owner", itUserB: "beto"}, nicknames)

	profiles, err := NewParticipantProfileProvider(pool).GetProfiles(context.Background(), []string{itOwner, itMissing})
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	assert.Equal(t, "owner", profiles[itOwner].Nickname)
	assert.Equal(t, "Colombia", profiles[itOwner].Country)
	assert.Equal(t, "Bogota", profiles[itOwner].City)
	assert.Equal(t, "UNAL", profiles[itOwner].Institution)

	teams, err := NewTeamDisplayProvider(pool).GetDisplays(context.Background(), []string{itTeam1, itTeam2, itMissing})
	require.NoError(t, err)
	require.Len(t, teams, 2)
	assert.Equal(t, "Team One", teams[itTeam1].Name)

	for _, empty := range []func() (int, error){
		func() (int, error) {
			m, err := NewParticipantNicknameProvider(pool).GetNicknamesByIDs(context.Background(), nil)
			return len(m), err
		},
		func() (int, error) {
			m, err := NewParticipantProfileProvider(pool).GetProfiles(context.Background(), nil)
			return len(m), err
		},
		func() (int, error) {
			m, err := NewTeamDisplayProvider(pool).GetDisplays(context.Background(), nil)
			return len(m), err
		},
	} {
		n, err := empty()
		require.NoError(t, err)
		assert.Equal(t, 0, n)
	}
}

func TestContestRejudgeProvider_ContestProblemMembershipAndGroupLead(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedWithContests(t, pool)
	pgtest.SeedContestProblem(t, pool, "dddddddd-0000-0000-0000-000000000001", itContest1, itProblem1, 1)
	pgtest.SeedGroupMember(t, pool, "77770000-0000-0000-0000-0000000000b1", itGroupA, itOwner, "LEAD")
	pgtest.SeedGroupMember(t, pool, "77770000-0000-0000-0000-0000000000b2", itGroupA, itUserB, "MEMBER")
	provider := NewContestRejudgeProvider(pool)

	info, err := provider.GetContestForRejudge(context.Background(), itContest1)
	require.NoError(t, err)
	assert.Equal(t, itContest1, info.ID)
	assert.Equal(t, itOwner, info.OwnerID)
	require.NotNil(t, info.GroupID)
	assert.Equal(t, itGroupA, *info.GroupID)
	assert.True(t, info.EndTime.After(info.StartTime))

	_, err = provider.GetContestForRejudge(context.Background(), itMissing)
	itRequireCode(t, err, domainContest.ErrCodeContestNotFound)

	in, err := provider.IsProblemInContest(context.Background(), itContest1, itProblem1)
	require.NoError(t, err)
	assert.True(t, in)
	out, err := provider.IsProblemInContest(context.Background(), itContest1, itProblem2)
	require.NoError(t, err)
	assert.False(t, out)

	lead, err := provider.IsLeadOfGroup(context.Background(), itOwner, itGroupA)
	require.NoError(t, err)
	assert.True(t, lead)
	member, err := provider.IsLeadOfGroup(context.Background(), itUserB, itGroupA)
	require.NoError(t, err)
	assert.False(t, member)
}

func TestStandingsSubmissionProvider_ListByContest_OnlyJudgedOldestFirst(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedWithContests(t, pool)
	t0 := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	add := func(id, user, status string, minute int) {
		pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: id, ProblemID: itProblem1, UserID: user, ContestID: itContest1, Status: status, SubmittedAt: t0.Add(time.Duration(minute) * time.Minute)})
	}
	add("late", itUserB, "ACCEPTED", 30)
	add("early", itOwner, "WRONG_ANSWER", 10)
	add("pending", itUserB, "PENDING", 20)
	add("running", itUserB, "RUNNING", 25)
	pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: "elsewhere", ProblemID: itProblem1, UserID: itOwner, ContestID: itContest2, Status: "ACCEPTED", SubmittedAt: t0})

	got, err := NewStandingsSubmissionProvider(pool).ListByContest(context.Background(), itContest1)

	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, itOwner, got[0].UserID)
	assert.Equal(t, "WRONG_ANSWER", got[0].Status)
	assert.Equal(t, itProblem1, got[0].ProblemID)
	assert.Equal(t, itUserB, got[1].UserID)
	assert.True(t, t0.Add(30*time.Minute).Equal(got[1].SubmittedAt))
}

func TestTeamSelectionChecker_IsUserSelectedInAnyTeam(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedWithContests(t, pool)
	itSeedTeams(t, pool)
	itSeedTeamEntry(t, pool, itPart1, itContest1, itTeam1, itOwner)
	checker := NewTeamSelectionChecker(pool)

	selected, err := checker.IsUserSelectedInAnyTeam(context.Background(), itContest1, itOwner)
	require.NoError(t, err)
	assert.True(t, selected)

	notSelected, err := checker.IsUserSelectedInAnyTeam(context.Background(), itContest1, itUserC)
	require.NoError(t, err)
	assert.False(t, notSelected)

	otherContest, err := checker.IsUserSelectedInAnyTeam(context.Background(), itContest2, itOwner)
	require.NoError(t, err)
	assert.False(t, otherContest)
}
