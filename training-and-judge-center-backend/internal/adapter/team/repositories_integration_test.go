//go:build integration

package team

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/internal/testutil/pgtest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestMain(m *testing.M) {
	os.Exit(pgtest.Main(m))
}

var itNow = time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)

const (
	itUserA   = "11111111-1111-1111-1111-111111111111"
	itUserB   = "22222222-2222-2222-2222-222222222222"
	itUserC   = "33333333-3333-3333-3333-333333333333"
	itTeam1   = "eeeeeeee-0000-0000-0000-000000000001"
	itTeam2   = "eeeeeeee-0000-0000-0000-000000000002"
	itMissing = "99999999-9999-9999-9999-999999999999"
	itMember  = "66660000-0000-0000-0000-00000000000"
	itInv     = "aaaa2222-0000-0000-0000-00000000000"
)

func itSeedUsers(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	pgtest.SeedUser(t, pool, itUserA, "ana")
	pgtest.SeedUser(t, pool, itUserB, "beto")
	pgtest.SeedUser(t, pool, itUserC, "carla")
}

func itTeam(t *testing.T, id, name string) *domainTeam.Team {
	t.Helper()
	teamName, err := domainTeam.NewTeamName(name)
	require.NoError(t, err)
	team, err := domainTeam.NewTeam(id, teamName, shared.RestoreUserID(itUserA), itNow)
	require.NoError(t, err)
	return team
}

func itSaveTeam(t *testing.T, pool *pgxpool.Pool, team *domainTeam.Team) {
	t.Helper()
	require.NoError(t, NewRepository(pool).Save(context.Background(), team))
}

func itRequireCode(t *testing.T, err error, code string) {
	t.Helper()
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, code, appErr.Code)
}

func TestRepository_SaveAndFindByID_RoundTrips(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedUsers(t, pool)
	itSaveTeam(t, pool, itTeam(t, itTeam1, "Red Dragons"))

	got, err := NewRepository(pool).FindByID(context.Background(), itTeam1)

	require.NoError(t, err)
	assert.Equal(t, "Red Dragons", got.Name().Value())
	assert.Equal(t, itUserA, got.CreatedBy().Value())
	assert.True(t, itNow.Equal(got.CreatedAt()))

	_, err = NewRepository(pool).FindByID(context.Background(), itMissing)
	itRequireCode(t, err, domainTeam.ErrCodeTeamNotFound)
	_, err = NewRepository(pool).FindByID(context.Background(), "not-a-uuid")
	itRequireCode(t, err, domainTeam.ErrCodeTeamNotFound)
}

func TestRepository_Save_DuplicateNameIsCaseInsensitiveConflict(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedUsers(t, pool)
	itSaveTeam(t, pool, itTeam(t, itTeam1, "Red Dragons"))

	err := NewRepository(pool).Save(context.Background(), itTeam(t, itTeam2, "RED DRAGONS"))

	itRequireCode(t, err, domainTeam.ErrCodeTeamNameExists)
}

func TestRepository_UpdateName_PersistsAndRejectsTakenOrMissing(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedUsers(t, pool)
	itSaveTeam(t, pool, itTeam(t, itTeam1, "Red Dragons"))
	itSaveTeam(t, pool, itTeam(t, itTeam2, "Blue Owls"))
	repo := NewRepository(pool)

	renamed := itTeam(t, itTeam1, "Green Foxes")
	require.NoError(t, repo.UpdateName(context.Background(), renamed))
	got, err := repo.FindByID(context.Background(), itTeam1)
	require.NoError(t, err)
	assert.Equal(t, "Green Foxes", got.Name().Value())

	itRequireCode(t, repo.UpdateName(context.Background(), itTeam(t, itTeam1, "Blue Owls")), domainTeam.ErrCodeTeamNameExists)
	itRequireCode(t, repo.UpdateName(context.Background(), itTeam(t, itMissing, "Ghosts")), domainTeam.ErrCodeTeamNotFound)
}

func TestRepository_FindByIDsAndExistsByName(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedUsers(t, pool)
	itSaveTeam(t, pool, itTeam(t, itTeam1, "Red Dragons"))
	itSaveTeam(t, pool, itTeam(t, itTeam2, "Blue Owls"))
	repo := NewRepository(pool)

	got, err := repo.FindByIDs(context.Background(), []string{itTeam1, itTeam2, itMissing})
	require.NoError(t, err)
	assert.Len(t, got, 2)

	empty, err := repo.FindByIDs(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
	assert.NotNil(t, empty)

	name, err := domainTeam.NewTeamName("red dragons")
	require.NoError(t, err)
	exists, err := repo.ExistsByName(context.Background(), name)
	require.NoError(t, err)
	assert.True(t, exists)
	other, err := domainTeam.NewTeamName("Yellow Bees")
	require.NoError(t, err)
	exists, err = repo.ExistsByName(context.Background(), other)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRepository_LockByID_LocksAnExistingTeamAndReportsAMissingOne(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedUsers(t, pool)
	itSaveTeam(t, pool, itTeam(t, itTeam1, "Red Dragons"))
	repo := NewRepository(pool)
	tm := postgres.NewTransactionManager(pool)

	err := tm.WithTx(context.Background(), func(txCtx context.Context) error {
		return repo.LockByID(txCtx, itTeam1)
	})
	require.NoError(t, err)

	err = tm.WithTx(context.Background(), func(txCtx context.Context) error {
		return repo.LockByID(txCtx, itMissing)
	})
	itRequireCode(t, err, domainTeam.ErrCodeTeamNotFound)
}

func itTeamMember(t *testing.T, id, teamID, userID string, at time.Time) *domainTeam.TeamMember {
	t.Helper()
	m, err := domainTeam.NewTeamMember(id, teamID, shared.RestoreUserID(userID), at)
	require.NoError(t, err)
	return m
}

func TestMemberRepository_SaveFindAndDelete(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedUsers(t, pool)
	itSaveTeam(t, pool, itTeam(t, itTeam1, "Red Dragons"))
	repo := NewMemberRepository(pool)
	require.NoError(t, repo.Save(context.Background(), itTeamMember(t, itMember+"1", itTeam1, itUserB, itNow)))

	got, err := repo.FindByTeamAndUser(context.Background(), itTeam1, shared.RestoreUserID(itUserB))
	require.NoError(t, err)
	assert.Equal(t, itMember+"1", got.ID())
	assert.True(t, itNow.Equal(got.JoinedAt()))

	_, err = repo.FindByTeamAndUser(context.Background(), itTeam1, shared.RestoreUserID(itUserC))
	itRequireCode(t, err, domainTeam.ErrCodeNotTeamMember)

	require.NoError(t, repo.DeleteByTeamAndUser(context.Background(), itTeam1, shared.RestoreUserID(itUserB)))
	itRequireCode(t, repo.DeleteByTeamAndUser(context.Background(), itTeam1, shared.RestoreUserID(itUserB)), domainTeam.ErrCodeNotTeamMember)
}

func TestMemberRepository_Save_SecondMembershipReturnsAlreadyOnTeam(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedUsers(t, pool)
	itSaveTeam(t, pool, itTeam(t, itTeam1, "Red Dragons"))
	repo := NewMemberRepository(pool)
	require.NoError(t, repo.Save(context.Background(), itTeamMember(t, itMember+"1", itTeam1, itUserB, itNow)))

	err := repo.Save(context.Background(), itTeamMember(t, itMember+"2", itTeam1, itUserB, itNow))

	itRequireCode(t, err, domainTeam.ErrCodeMemberAlreadyOnTeam)
}

func TestMemberRepository_ListsAndCounts(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedUsers(t, pool)
	itSaveTeam(t, pool, itTeam(t, itTeam1, "Red Dragons"))
	itSaveTeam(t, pool, itTeam(t, itTeam2, "Blue Owls"))
	repo := NewMemberRepository(pool)
	require.NoError(t, repo.Save(context.Background(), itTeamMember(t, itMember+"1", itTeam1, itUserA, itNow)))
	require.NoError(t, repo.Save(context.Background(), itTeamMember(t, itMember+"2", itTeam1, itUserB, itNow.Add(time.Hour))))
	require.NoError(t, repo.Save(context.Background(), itTeamMember(t, itMember+"3", itTeam2, itUserB, itNow.Add(2*time.Hour))))

	byTeam, err := repo.FindByTeam(context.Background(), itTeam1)
	require.NoError(t, err)
	assert.Len(t, byTeam, 2)

	byUser, err := repo.FindByUser(context.Background(), shared.RestoreUserID(itUserB))
	require.NoError(t, err)
	require.Len(t, byUser, 2)
	assert.Equal(t, itTeam2, byUser[0].TeamID(), "most recently joined first")

	counts, err := repo.BulkCountByTeams(context.Background(), []string{itTeam1, itTeam2, itMissing})
	require.NoError(t, err)
	assert.Equal(t, map[string]int{itTeam1: 2, itTeam2: 1}, counts)

	empty, err := repo.BulkCountByTeams(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, empty)

	none, err := repo.FindByTeam(context.Background(), itMissing)
	require.NoError(t, err)
	assert.Empty(t, none)
}

func itInvitation(t *testing.T, id, teamID, invitee string, at time.Time) *domainTeam.TeamInvitation {
	t.Helper()
	inv, err := domainTeam.NewTeamInvitation(id, teamID, shared.RestoreUserID(invitee), shared.RestoreUserID(itUserA), at)
	require.NoError(t, err)
	return inv
}

func TestInvitationRepository_SaveFindAndDelete(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedUsers(t, pool)
	itSaveTeam(t, pool, itTeam(t, itTeam1, "Red Dragons"))
	repo := NewInvitationRepository(pool)
	require.NoError(t, repo.Save(context.Background(), itInvitation(t, itInv+"1", itTeam1, itUserB, itNow)))

	byID, err := repo.FindByID(context.Background(), itInv+"1")
	require.NoError(t, err)
	assert.Equal(t, itUserB, byID.InviteeID().Value())
	assert.Equal(t, itUserA, byID.InvitedBy().Value())

	byPair, err := repo.FindByTeamAndInvitee(context.Background(), itTeam1, shared.RestoreUserID(itUserB))
	require.NoError(t, err)
	assert.Equal(t, itInv+"1", byPair.ID())

	_, err = repo.FindByID(context.Background(), itMissing)
	itRequireCode(t, err, domainTeam.ErrCodeInvitationNotFound)
	_, err = repo.FindByTeamAndInvitee(context.Background(), itTeam1, shared.RestoreUserID(itUserC))
	itRequireCode(t, err, domainTeam.ErrCodeInvitationNotFound)

	require.NoError(t, repo.Delete(context.Background(), itInv+"1"))
	itRequireCode(t, repo.Delete(context.Background(), itInv+"1"), domainTeam.ErrCodeInvitationNotFound)
}

func TestInvitationRepository_Save_SecondInvitationToTheSameUserIsRefused(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedUsers(t, pool)
	itSaveTeam(t, pool, itTeam(t, itTeam1, "Red Dragons"))
	repo := NewInvitationRepository(pool)
	require.NoError(t, repo.Save(context.Background(), itInvitation(t, itInv+"1", itTeam1, itUserB, itNow)))

	err := repo.Save(context.Background(), itInvitation(t, itInv+"2", itTeam1, itUserB, itNow))

	itRequireCode(t, err, domainTeam.ErrCodeAlreadyInvited)
}

func TestInvitationRepository_FindByTeamAndInvitee_NewestFirst(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedUsers(t, pool)
	itSaveTeam(t, pool, itTeam(t, itTeam1, "Red Dragons"))
	itSaveTeam(t, pool, itTeam(t, itTeam2, "Blue Owls"))
	repo := NewInvitationRepository(pool)
	require.NoError(t, repo.Save(context.Background(), itInvitation(t, itInv+"1", itTeam1, itUserB, itNow)))
	require.NoError(t, repo.Save(context.Background(), itInvitation(t, itInv+"2", itTeam1, itUserC, itNow.Add(time.Hour))))
	require.NoError(t, repo.Save(context.Background(), itInvitation(t, itInv+"3", itTeam2, itUserB, itNow.Add(2*time.Hour))))

	byTeam, err := repo.FindByTeam(context.Background(), itTeam1)
	require.NoError(t, err)
	require.Len(t, byTeam, 2)
	assert.Equal(t, itInv+"2", byTeam[0].ID())

	byInvitee, err := repo.FindByInvitee(context.Background(), shared.RestoreUserID(itUserB))
	require.NoError(t, err)
	require.Len(t, byInvitee, 2)
	assert.Equal(t, itInv+"3", byInvitee[0].ID())

	none, err := repo.FindByInvitee(context.Background(), shared.RestoreUserID(itMissing))
	require.NoError(t, err)
	assert.Empty(t, none)
}
