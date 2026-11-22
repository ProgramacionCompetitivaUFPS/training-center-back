//go:build integration

package team

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/internal/testutil/pgtest"
)

const (
	itGroup    = "bbbbbbbb-0000-0000-0000-00000000000a"
	itContest1 = "cccccccc-0000-0000-0000-000000000001"
	itContest2 = "cccccccc-0000-0000-0000-000000000002"
)

func itSeedTeamsAndGroup(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	itSeedUsers(t, pool)
	pgtest.SeedGroup(t, pool, itGroup, "Group A", itUserA)
	itSaveTeam(t, pool, itTeam(t, itTeam1, "Red Dragons"))
	itSaveTeam(t, pool, itTeam(t, itTeam2, "Blue Owls"))
}

func itSeedContest(t *testing.T, pool *pgxpool.Pool, id, name string, start, end time.Time) {
	t.Helper()
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: id, GroupID: itGroup, OwnerID: itUserA, Name: name, Start: start, End: end})
}

func TestContestParticipationChecker_IsUserInActiveContestForTeam(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedTeamsAndGroup(t, pool)
	now := time.Now()
	itSeedContest(t, pool, itContest1, "Running", now.Add(-time.Hour), now.Add(time.Hour))
	itSeedContest(t, pool, itContest2, "Later", now.Add(24*time.Hour), now.Add(26*time.Hour))
	pgtest.SeedTeamParticipation(t, pool, "99990000-0000-0000-0000-000000000001", itContest1, itTeam1, itUserA, itUserB)
	pgtest.SeedTeamParticipation(t, pool, "99990000-0000-0000-0000-000000000002", itContest2, itTeam2, itUserA, itUserB)
	checker := NewContestParticipationChecker(pool)

	selected, err := checker.IsUserInActiveContestForTeam(context.Background(), itUserA, itTeam1)
	require.NoError(t, err)
	assert.True(t, selected)

	notSelected, err := checker.IsUserInActiveContestForTeam(context.Background(), itUserC, itTeam1)
	require.NoError(t, err)
	assert.False(t, notSelected)

	notStarted, err := checker.IsUserInActiveContestForTeam(context.Background(), itUserA, itTeam2)
	require.NoError(t, err)
	assert.False(t, notStarted)
}

func TestContestProvider_GetContestInfo_DerivesTheStatusFromTheClock(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedTeamsAndGroup(t, pool)
	now := time.Now()
	itSeedContest(t, pool, itContest1, "Running", now.Add(-time.Hour), now.Add(time.Hour))
	itSeedContest(t, pool, itContest2, "Later", now.Add(24*time.Hour), now.Add(26*time.Hour))
	itSeedContest(t, pool, "cccccccc-0000-0000-0000-000000000003", "Done", now.Add(-48*time.Hour), now.Add(-24*time.Hour))
	provider := NewContestProvider(pool)

	running, err := provider.GetContestInfo(context.Background(), itContest1)
	require.NoError(t, err)
	assert.Equal(t, "ACTIVE", running.Status)
	assert.Equal(t, itGroup, running.GroupID)
	assert.True(t, running.GroupIsVisible)
	assert.Equal(t, "INDIVIDUAL", running.ParticipationMode)
	assert.Equal(t, 2, running.TeamSizeMin)
	assert.Equal(t, 5, running.TeamSizeMax)

	later, err := provider.GetContestInfo(context.Background(), itContest2)
	require.NoError(t, err)
	assert.Equal(t, "SCHEDULED", later.Status)

	done, err := provider.GetContestInfo(context.Background(), "cccccccc-0000-0000-0000-000000000003")
	require.NoError(t, err)
	assert.Equal(t, "FINISHED", done.Status)

	_, err = provider.GetContestInfo(context.Background(), itMissing)
	itRequireCode(t, err, "CONTEST_NOT_FOUND")
}

func TestGroupMemberCheckerAndIndividualRegistrationChecker(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedTeamsAndGroup(t, pool)
	now := time.Now()
	itSeedContest(t, pool, itContest1, "Later", now.Add(24*time.Hour), now.Add(26*time.Hour))
	pgtest.SeedGroupMember(t, pool, "77770000-0000-0000-0000-0000000000a1", itGroup, itUserA, "LEAD")
	pgtest.SeedRegistration(t, pool, "ffffffff-0000-0000-0000-000000000001", itContest1, itUserB)
	ids := []string{itUserA, itUserB, itUserC}

	members, err := NewGroupMemberChecker(pool).AreUsersGroupMembers(context.Background(), itGroup, ids)
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{itUserA: true, itUserB: false, itUserC: false}, members)

	registered, err := NewIndividualRegistrationChecker(pool).AreUsersRegisteredIndividually(context.Background(), itContest1, ids)
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{itUserA: false, itUserB: true, itUserC: false}, registered)

	noMembers, err := NewGroupMemberChecker(pool).AreUsersGroupMembers(context.Background(), itGroup, nil)
	require.NoError(t, err)
	assert.Empty(t, noMembers)
	noRegistered, err := NewIndividualRegistrationChecker(pool).AreUsersRegisteredIndividually(context.Background(), itContest1, nil)
	require.NoError(t, err)
	assert.Empty(t, noRegistered)
}

func TestUserProvider_GetDisplayGetDisplaysAndFindByNickname(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedUsers(t, pool)
	provider := NewUserProvider(pool)

	one, err := provider.GetDisplay(context.Background(), itUserA)
	require.NoError(t, err)
	assert.Equal(t, "ana", one.Nickname)
	_, err = provider.GetDisplay(context.Background(), itMissing)
	itRequireCode(t, err, domainTeam.ErrCodeUserNotFound)

	many, err := provider.GetDisplays(context.Background(), []string{itUserA, itUserB, itMissing})
	require.NoError(t, err)
	assert.Len(t, many, 2)
	empty, err := provider.GetDisplays(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, empty)

	byNick, err := provider.FindByNickname(context.Background(), "BETO")
	require.NoError(t, err)
	assert.Equal(t, itUserB, byNick.ID)
	_, err = provider.FindByNickname(context.Background(), "nobody")
	itRequireCode(t, err, domainTeam.ErrCodeUserNotFound)
}

func TestScheduledParticipationCleaner_ReducesOrCancelsAndReportsWhatWasCancelled(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedTeamsAndGroup(t, pool)
	now := time.Now()
	itSeedContest(t, pool, itContest1, "Alpha Cup", now.Add(24*time.Hour), now.Add(26*time.Hour))
	itSeedContest(t, pool, itContest2, "Beta Cup", now.Add(24*time.Hour), now.Add(26*time.Hour))
	itSeedContest(t, pool, "cccccccc-0000-0000-0000-000000000003", "Running Cup", now.Add(-time.Hour), now.Add(time.Hour))
	pgtest.SeedTeamParticipation(t, pool, "99990000-0000-0000-0000-000000000001", itContest1, itTeam1, itUserA, itUserB, itUserC)
	pgtest.SeedTeamParticipation(t, pool, "99990000-0000-0000-0000-000000000002", itContest2, itTeam1, itUserB, itUserC)
	pgtest.SeedTeamParticipation(t, pool, "99990000-0000-0000-0000-000000000003", "cccccccc-0000-0000-0000-000000000003", itTeam1, itUserA, itUserC)
	cleaner := NewScheduledParticipationCleaner(pool)

	atRisk, err := cleaner.RegistrationsLosingMinimum(context.Background(), itTeam1, itUserC)
	require.NoError(t, err)
	require.Len(t, atRisk, 1)
	assert.Equal(t, "Beta Cup", atRisk[0].ContestName)
	assert.Equal(t, itContest2, atRisk[0].ContestID)
	assert.Equal(t, itGroup, atRisk[0].GroupID)

	cancelled, err := cleaner.RemoveUserFromScheduledParticipations(context.Background(), itTeam1, itUserC)
	require.NoError(t, err)
	require.Len(t, cancelled, 1)
	assert.Equal(t, "Beta Cup", cancelled[0].ContestName)

	var reduced []string
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT selected_members FROM contest_team_participants WHERE id = '99990000-0000-0000-0000-000000000001'`).Scan(&reduced))
	assert.ElementsMatch(t, []string{itUserA, itUserB}, reduced)
	var left int
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM contest_team_participants WHERE id = '99990000-0000-0000-0000-000000000002'`).Scan(&left))
	assert.Equal(t, 0, left)
	var running []string
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT selected_members FROM contest_team_participants WHERE id = '99990000-0000-0000-0000-000000000003'`).Scan(&running))
	assert.ElementsMatch(t, []string{itUserA, itUserC}, running, "a contest that already started is never touched")
}

func TestMembershipCleaner_RemoveUserFromAllTeams_ClearsMembershipsInvitationsAndScheduledSelections(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedTeamsAndGroup(t, pool)
	now := time.Now()
	itSeedContest(t, pool, itContest1, "Alpha Cup", now.Add(24*time.Hour), now.Add(26*time.Hour))
	member := func(id, team, user string) {
		require.NoError(t, NewMemberRepository(pool).Save(context.Background(), itTeamMember(t, id, team, user, itNow)))
	}
	member(itMember+"1", itTeam1, itUserA)
	member(itMember+"2", itTeam1, itUserB)
	member(itMember+"3", itTeam2, itUserB)
	require.NoError(t, NewInvitationRepository(pool).Save(context.Background(), itInvitation(t, itInv+"1", itTeam2, itUserB, itNow)))
	require.NoError(t, NewInvitationRepository(pool).Save(context.Background(), itInvitation(t, itInv+"2", itTeam2, itUserC, itNow)))
	pgtest.SeedTeamParticipation(t, pool, "99990000-0000-0000-0000-000000000001", itContest1, itTeam1, itUserA, itUserB, itUserC)

	require.NoError(t, NewMembershipCleaner(pool).RemoveUserFromAllTeams(context.Background(), itUserB))

	memberships, err := NewMemberRepository(pool).FindByUser(context.Background(), shared.RestoreUserID(itUserB))
	require.NoError(t, err)
	assert.Empty(t, memberships)
	invitations, err := NewInvitationRepository(pool).FindByInvitee(context.Background(), shared.RestoreUserID(itUserB))
	require.NoError(t, err)
	assert.Empty(t, invitations)
	others, err := NewInvitationRepository(pool).FindByInvitee(context.Background(), shared.RestoreUserID(itUserC))
	require.NoError(t, err)
	assert.Len(t, others, 1)
	var selected []string
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT selected_members FROM contest_team_participants WHERE id = '99990000-0000-0000-0000-000000000001'`).Scan(&selected))
	assert.ElementsMatch(t, []string{itUserA, itUserC}, selected)
	stillThere, err := NewMemberRepository(pool).FindByTeam(context.Background(), itTeam1)
	require.NoError(t, err)
	assert.Len(t, stillThere, 1)
}
