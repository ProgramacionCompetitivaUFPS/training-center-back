//go:build integration

package contest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/training-judge-center/backend/internal/adapter/postgres"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/testutil/pgtest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	itUserC = "33333333-3333-3333-3333-333333333333"
	itTeam1 = "eeeeeeee-0000-0000-0000-000000000001"
	itTeam2 = "eeeeeeee-0000-0000-0000-000000000002"
	itReg1  = "ffffffff-0000-0000-0000-000000000001"
	itReg2  = "ffffffff-0000-0000-0000-000000000002"
	itPart1 = "99990000-0000-0000-0000-000000000001"
	itPart2 = "99990000-0000-0000-0000-000000000002"
)

var itNow = time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)

func itSeedWithContests(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	itSeedBase(t, pool)
	pgtest.SeedUser(t, pool, itUserC, "carla")
	start := time.Now().Add(time.Hour)
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: itContest1, GroupID: itGroupA, OwnerID: itOwner, Start: start, End: start.Add(time.Hour)})
	pgtest.SeedContest(t, pool, pgtest.Contest{ID: itContest2, GroupID: itGroupA, OwnerID: itOwner, Start: start, End: start.Add(time.Hour)})
}

func itRegistration(t *testing.T, id, contestID, userID string, at time.Time) *domainContest.ContestRegistration {
	t.Helper()
	reg, err := domainContest.NewContestRegistration(id, contestID, userID, at)
	require.NoError(t, err)
	return reg
}

func TestRegistrationRepository_SaveAndFind_RoundTrips(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedWithContests(t, pool)
	repo := NewRegistrationRepository(pool)

	require.NoError(t, repo.Save(context.Background(), itRegistration(t, itReg1, itContest1, itOwner, itNow)))

	got, err := repo.FindByContestAndUser(context.Background(), itContest1, itOwner)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, itReg1, got.ID())
	assert.True(t, itNow.Equal(got.RegisteredAt()))

	missing, err := repo.FindByContestAndUser(context.Background(), itContest1, itUserB)
	require.NoError(t, err)
	assert.Nil(t, missing)

	malformed, err := repo.FindByContestAndUser(context.Background(), "not-a-uuid", itOwner)
	require.NoError(t, err)
	assert.Nil(t, malformed)
}

func TestRegistrationRepository_Save_RegisteringTwiceIsAnIdempotentNoOp(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedWithContests(t, pool)
	repo := NewRegistrationRepository(pool)
	require.NoError(t, repo.Save(context.Background(), itRegistration(t, itReg1, itContest1, itOwner, itNow)))

	err := repo.Save(context.Background(), itRegistration(t, itReg2, itContest1, itOwner, itNow.Add(time.Hour)))

	require.NoError(t, err)
	got, err := repo.FindByContestAndUser(context.Background(), itContest1, itOwner)
	require.NoError(t, err)
	assert.Equal(t, itReg1, got.ID())
	count, err := repo.CountByContest(context.Background(), itContest1)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestRegistrationRepository_Save_UnknownContestReturnsInternalNotRawError(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedWithContests(t, pool)

	err := NewRegistrationRepository(pool).Save(context.Background(), itRegistration(t, itReg1, itMissing, itOwner, itNow))

	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.KindInternal, appErr.Kind)
}

func TestRegistrationRepository_Delete_RemovesAndThenReportsNotRegistered(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedWithContests(t, pool)
	repo := NewRegistrationRepository(pool)
	require.NoError(t, repo.Save(context.Background(), itRegistration(t, itReg1, itContest1, itOwner, itNow)))

	require.NoError(t, repo.Delete(context.Background(), itContest1, itOwner))

	exists, err := repo.ExistsByContestAndUser(context.Background(), itContest1, itOwner)
	require.NoError(t, err)
	assert.False(t, exists)
	itRequireCode(t, repo.Delete(context.Background(), itContest1, itOwner), domainContest.ErrCodeNotRegistered)
}

func TestRegistrationRepository_ListByContest_OrdersByRegistrationAndPaginates(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedWithContests(t, pool)
	repo := NewRegistrationRepository(pool)
	require.NoError(t, repo.Save(context.Background(), itRegistration(t, itReg2, itContest1, itUserB, itNow.Add(time.Hour))))
	require.NoError(t, repo.Save(context.Background(), itRegistration(t, itReg1, itContest1, itOwner, itNow)))
	require.NoError(t, repo.Save(context.Background(), itRegistration(t, "ffffffff-0000-0000-0000-000000000003", itContest1, itUserC, itNow.Add(2*time.Hour))))

	first, total, err := repo.ListByContest(context.Background(), itContest1, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, first, 2)
	assert.Equal(t, itOwner, first[0].UserID())
	assert.Equal(t, itUserB, first[1].UserID())

	second, _, err := repo.ListByContest(context.Background(), itContest1, 2, 2)
	require.NoError(t, err)
	require.Len(t, second, 1)
	assert.Equal(t, itUserC, second[0].UserID())

	none, total, err := repo.ListByContest(context.Background(), itContest2, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, none)
}

func itSeedTeamEntry(t *testing.T, pool *pgxpool.Pool, partID, contestID, teamID string, members ...string) {
	t.Helper()
	p, err := domainContest.NewContestTeamParticipation(partID, contestID, teamID, members, itNow)
	require.NoError(t, err)
	require.NoError(t, NewTeamParticipationRepository(pool).Save(context.Background(), p))
}

func itSeedTeams(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	pgtest.SeedTeam(t, pool, itTeam1, "Team One", itOwner)
	pgtest.SeedTeam(t, pool, itTeam2, "Team Two", itUserB)
	pgtest.SeedTeamMember(t, pool, "77770000-0000-0000-0000-000000000001", itTeam1, itOwner)
	pgtest.SeedTeamMember(t, pool, "77770000-0000-0000-0000-000000000002", itTeam1, itUserC)
	pgtest.SeedTeamMember(t, pool, "77770000-0000-0000-0000-000000000003", itTeam2, itUserB)
}

func TestRegistrationRepository_Participation_CountsIndividualsAndTeamsAsOneEach(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedWithContests(t, pool)
	itSeedTeams(t, pool)
	repo := NewRegistrationRepository(pool)
	require.NoError(t, repo.Save(context.Background(), itRegistration(t, itReg1, itContest1, itUserB, itNow)))
	itSeedTeamEntry(t, pool, itPart1, itContest1, itTeam1, itOwner, itUserC)

	count, err := repo.CountByContest(context.Background(), itContest1)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "one individual plus one team, whatever its roster size")

	bulk, err := repo.CountByContestBulk(context.Background(), []string{itContest1, itContest2})
	require.NoError(t, err)
	assert.Equal(t, map[string]int{itContest1: 2, itContest2: 0}, bulk)

	empty, err := repo.CountByContestBulk(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestRegistrationRepository_ExistsByContestAndUserOrTeam_OnlySelectedTeamMembersCount(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedWithContests(t, pool)
	itSeedTeams(t, pool)
	repo := NewRegistrationRepository(pool)
	require.NoError(t, repo.Save(context.Background(), itRegistration(t, itReg1, itContest1, itUserB, itNow)))
	itSeedTeamEntry(t, pool, itPart1, itContest1, itTeam1, itOwner)

	tests := []struct {
		name string
		user string
		want bool
	}{
		{"individually registered", itUserB, true},
		{"selected member of a registered team", itOwner, true},
		{"team member who was not selected", itUserC, false},
		{"not registered at all", "99999999-9999-9999-9999-999999999998", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.ExistsByContestAndUserOrTeam(context.Background(), itContest1, tt.user)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	bulk, err := repo.ExistsByUserBulk(context.Background(), []string{itContest1, itContest2}, itOwner)
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{itContest1: true, itContest2: false}, bulk)

	empty, err := repo.ExistsByUserBulk(context.Background(), nil, itOwner)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestTeamParticipationRepository_SaveFindUpdateDelete(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedWithContests(t, pool)
	itSeedTeams(t, pool)
	repo := NewTeamParticipationRepository(pool)
	itSeedTeamEntry(t, pool, itPart1, itContest1, itTeam1, itOwner, itUserC)

	got, err := repo.FindByContestAndTeam(context.Background(), itContest1, itTeam1)
	require.NoError(t, err)
	assert.Equal(t, itPart1, got.ID())
	assert.ElementsMatch(t, []string{itOwner, itUserC}, got.SelectedMembers())

	got.UpdateSelectedMembers([]string{itUserC})
	require.NoError(t, repo.Update(context.Background(), got))
	updated, err := repo.FindByContestAndTeam(context.Background(), itContest1, itTeam1)
	require.NoError(t, err)
	assert.Equal(t, []string{itUserC}, updated.SelectedMembers())

	require.NoError(t, repo.Delete(context.Background(), itContest1, itTeam1))
	_, err = repo.FindByContestAndTeam(context.Background(), itContest1, itTeam1)
	itRequireCode(t, err, domainContest.ErrCodeTeamRegistrationNotFound)
	itRequireCode(t, repo.Delete(context.Background(), itContest1, itTeam1), domainContest.ErrCodeTeamRegistrationNotFound)
	itRequireCode(t, repo.Update(context.Background(), got), domainContest.ErrCodeTeamRegistrationNotFound)
}

func TestTeamParticipationRepository_Save_SameTeamTwiceReturnsConflict(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedWithContests(t, pool)
	itSeedTeams(t, pool)
	itSeedTeamEntry(t, pool, itPart1, itContest1, itTeam1, itOwner)
	again, err := domainContest.NewContestTeamParticipation(itPart2, itContest1, itTeam1, []string{itUserC}, itNow)
	require.NoError(t, err)

	err = NewTeamParticipationRepository(pool).Save(context.Background(), again)

	itRequireCode(t, err, domainContest.ErrCodeTeamAlreadyRegistered)
}

func TestTeamParticipationRepository_Save_EmptySelectionReturnsInternalNotRawError(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedWithContests(t, pool)
	itSeedTeams(t, pool)
	empty, err := domainContest.NewContestTeamParticipation(itPart1, itContest1, itTeam1, []string{}, itNow)
	require.NoError(t, err)

	err = NewTeamParticipationRepository(pool).Save(context.Background(), empty)

	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.KindInternal, appErr.Kind)
}

func TestTeamParticipationRepository_ListAndSelectedMembers(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedWithContests(t, pool)
	itSeedTeams(t, pool)
	repo := NewTeamParticipationRepository(pool)
	itSeedTeamEntry(t, pool, itPart1, itContest1, itTeam1, itOwner, itUserC)
	itSeedTeamEntry(t, pool, itPart2, itContest1, itTeam2, itUserB)

	page, total, err := repo.List(context.Background(), itContest1, 1, 1)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, page, 1)

	none, total, err := repo.List(context.Background(), itContest2, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, none)

	byTeam, err := repo.ListSelectedMembersByContest(context.Background(), itContest1)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{itOwner, itUserC}, byTeam[itTeam1])
	assert.Equal(t, []string{itUserB}, byTeam[itTeam2])
}

func TestTeamParticipationRepository_AreUsersSelectedInAnyTeam(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedWithContests(t, pool)
	itSeedTeams(t, pool)
	repo := NewTeamParticipationRepository(pool)
	itSeedTeamEntry(t, pool, itPart1, itContest1, itTeam1, itOwner)
	itSeedTeamEntry(t, pool, itPart2, itContest1, itTeam2, itUserB)

	all, err := repo.AreUsersSelectedInAnyTeam(context.Background(), itContest1, []string{itOwner, itUserB, itUserC}, "")
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{itOwner: true, itUserB: true, itUserC: false}, all)

	excluding, err := repo.AreUsersSelectedInAnyTeam(context.Background(), itContest1, []string{itOwner, itUserB}, itTeam1)
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{itOwner: false, itUserB: true}, excluding)

	none, err := repo.AreUsersSelectedInAnyTeam(context.Background(), itContest1, nil, "")
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestRegistrationLocker_Lock_SecondTransactionOnTheSameContestTimesOutAsBusy(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedWithContests(t, pool)
	locker := NewRegistrationLocker(pool)
	tm := postgres.NewTransactionManager(pool)
	locked := make(chan struct{})
	release := make(chan struct{})
	holderDone := make(chan error, 1)
	go func() {
		holderDone <- tm.WithTx(context.Background(), func(txCtx context.Context) error {
			if err := locker.Lock(txCtx, itContest1); err != nil {
				return err
			}
			close(locked)
			<-release
			return nil
		})
	}()
	<-locked

	err := tm.WithTx(context.Background(), func(txCtx context.Context) error {
		return locker.Lock(txCtx, itContest1)
	})

	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, domainContest.ErrCodeRegistrationBusy, appErr.Code)
	close(release)
	require.NoError(t, <-holderDone)

	err = tm.WithTx(context.Background(), func(txCtx context.Context) error {
		return locker.Lock(txCtx, itContest1)
	})
	assert.NoError(t, err, "the lock is released when the holding transaction ends")
}

func TestRegistrationLocker_Lock_DifferentContestsDoNotBlockEachOther(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedWithContests(t, pool)
	locker := NewRegistrationLocker(pool)
	tm := postgres.NewTransactionManager(pool)
	rollback := errors.New("stop")

	err := tm.WithTx(context.Background(), func(txCtx context.Context) error {
		require.NoError(t, locker.Lock(txCtx, itContest1))
		return tm.WithTx(context.Background(), func(otherCtx context.Context) error {
			require.NoError(t, locker.Lock(otherCtx, itContest2))
			return rollback
		})
	})

	require.ErrorIs(t, err, rollback)
}
