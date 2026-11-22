//go:build integration

package user

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainUser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/internal/testutil/pgtest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	requestID1 = "aaaaaaaa-0000-0000-0000-000000000001"
	requestID2 = "aaaaaaaa-0000-0000-0000-000000000002"
)

func TestEmailChangeRepository_SaveAndFindByID_RoundTrips(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedUser(t, pool, userSpec{id: userA, email: "ana@example.com", nickname: "ana"})
	repo := NewEmailChangeRepository(pool)
	newEmail, err := domainUser.NewEmail("nueva@example.com")
	require.NoError(t, err)
	req, err := domainUser.NewEmailChangeRequest(requestID1, userA, newEmail, "123456", integrationNow)
	require.NoError(t, err)

	require.NoError(t, repo.Save(context.Background(), req))

	got, err := repo.FindByID(context.Background(), requestID1)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, userA, got.UserID())
	assert.Equal(t, "nueva@example.com", got.NewEmail().String())
	assert.Equal(t, "123456", got.Code())
	assert.Equal(t, "PENDING", got.Status().String())
	assert.True(t, req.ExpiresAt().Equal(got.ExpiresAt()))
}

func TestEmailChangeRepository_FindByID_MissingReturnsNil(t *testing.T) {
	pool := pgtest.NewDB(t)

	got, err := NewEmailChangeRepository(pool).FindByID(context.Background(), requestID1)

	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestEmailChangeRepository_FindByCodeAndUserID_MatchesBothOrNeither(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedUser(t, pool, userSpec{id: userA, email: "ana@example.com", nickname: "ana"})
	repo := NewEmailChangeRepository(pool)
	newEmail, _ := domainUser.NewEmail("nueva@example.com")
	req, err := domainUser.NewEmailChangeRequest(requestID1, userA, newEmail, "123456", integrationNow)
	require.NoError(t, err)
	require.NoError(t, repo.Save(context.Background(), req))

	found, err := repo.FindByCodeAndUserID(context.Background(), "123456", userA)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, requestID1, found.ID())

	wrongCode, err := repo.FindByCodeAndUserID(context.Background(), "000000", userA)
	require.NoError(t, err)
	assert.Nil(t, wrongCode)

	wrongUser, err := repo.FindByCodeAndUserID(context.Background(), "123456", userB)
	require.NoError(t, err)
	assert.Nil(t, wrongUser)
}

func TestEmailChangeRepository_Update_PersistsUsedStatus(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedUser(t, pool, userSpec{id: userA, email: "ana@example.com", nickname: "ana"})
	repo := NewEmailChangeRepository(pool)
	newEmail, _ := domainUser.NewEmail("nueva@example.com")
	req, err := domainUser.NewEmailChangeRequest(requestID1, userA, newEmail, "123456", integrationNow)
	require.NoError(t, err)
	require.NoError(t, repo.Save(context.Background(), req))

	require.NoError(t, req.MarkAsUsed(integrationNow.Add(time.Minute)))
	require.NoError(t, repo.Update(context.Background(), req))

	got, err := repo.FindByID(context.Background(), requestID1)
	require.NoError(t, err)
	assert.Equal(t, "USED", got.Status().String())
	require.NotNil(t, got.UpdatedAt())
}

func TestEmailChangeRepository_InvalidatePendingByUserID_ExpiresOnlyPendingOfThatUser(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedUser(t, pool, userSpec{id: userA, email: "ana@example.com", nickname: "ana"})
	seedUser(t, pool, userSpec{id: userB, email: "beto@example.com", nickname: "beto"})
	repo := NewEmailChangeRepository(pool)
	newEmail, _ := domainUser.NewEmail("nueva@example.com")
	pendingA, _ := domainUser.NewEmailChangeRequest(requestID1, userA, newEmail, "111111", integrationNow)
	pendingB, _ := domainUser.NewEmailChangeRequest(requestID2, userB, newEmail, "222222", integrationNow)
	require.NoError(t, repo.Save(context.Background(), pendingA))
	require.NoError(t, repo.Save(context.Background(), pendingB))

	require.NoError(t, repo.InvalidatePendingByUserID(context.Background(), userA, integrationNow.Add(time.Minute)))

	gotA, err := repo.FindByID(context.Background(), requestID1)
	require.NoError(t, err)
	assert.Equal(t, "EXPIRED", gotA.Status().String())
	gotB, err := repo.FindByID(context.Background(), requestID2)
	require.NoError(t, err)
	assert.Equal(t, "PENDING", gotB.Status().String())
}

func TestPasswordRecoveryRepository_SaveFindAndUpdate(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedUser(t, pool, userSpec{id: userA, email: "ana@example.com", nickname: "ana"})
	repo := NewPasswordRecoveryRepository(pool)
	req, err := domainUser.NewPasswordRecoveryRequest(requestID1, userA, "654321", integrationNow)
	require.NoError(t, err)

	require.NoError(t, repo.Save(context.Background(), req))

	byID, err := repo.FindByID(context.Background(), requestID1)
	require.NoError(t, err)
	require.NotNil(t, byID)
	assert.Equal(t, "654321", byID.Code())
	assert.Equal(t, "PENDING", byID.Status().String())

	pending, err := repo.FindPendingByUserID(context.Background(), userA)
	require.NoError(t, err)
	require.NotNil(t, pending)
	assert.Equal(t, requestID1, pending.ID())

	require.NoError(t, req.MarkAsUsed(integrationNow.Add(time.Minute)))
	require.NoError(t, repo.Update(context.Background(), req))

	afterUse, err := repo.FindPendingByUserID(context.Background(), userA)
	require.NoError(t, err)
	assert.Nil(t, afterUse)
	used, err := repo.FindByID(context.Background(), requestID1)
	require.NoError(t, err)
	assert.Equal(t, "USED", used.Status().String())
}

func TestPasswordRecoveryRepository_FindMissingReturnsNil(t *testing.T) {
	pool := pgtest.NewDB(t)
	repo := NewPasswordRecoveryRepository(pool)

	byID, err := repo.FindByID(context.Background(), requestID1)
	require.NoError(t, err)
	assert.Nil(t, byID)

	pending, err := repo.FindPendingByUserID(context.Background(), userA)
	require.NoError(t, err)
	assert.Nil(t, pending)
}

func TestPasswordRecoveryRepository_InvalidatePendingByUserID_ExpiresPending(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedUser(t, pool, userSpec{id: userA, email: "ana@example.com", nickname: "ana"})
	repo := NewPasswordRecoveryRepository(pool)
	req, err := domainUser.NewPasswordRecoveryRequest(requestID1, userA, "654321", integrationNow)
	require.NoError(t, err)
	require.NoError(t, repo.Save(context.Background(), req))

	require.NoError(t, repo.InvalidatePendingByUserID(context.Background(), userA, integrationNow.Add(time.Minute)))

	got, err := repo.FindByID(context.Background(), requestID1)
	require.NoError(t, err)
	assert.Equal(t, "EXPIRED", got.Status().String())
}

func TestDeactivationRequestRepository_SaveAndFind(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedUser(t, pool, userSpec{id: userA, email: "ana@example.com", nickname: "ana"})
	repo := NewDeactivationRequestRepository(pool)
	req, err := domainUser.NewDeactivationRequest(requestID1, userA, "135790", integrationNow)
	require.NoError(t, err)

	require.NoError(t, repo.Save(context.Background(), req))

	byID, err := repo.FindByID(context.Background(), requestID1)
	require.NoError(t, err)
	require.NotNil(t, byID)
	assert.Equal(t, "135790", byID.VerificationCode())
	assert.Equal(t, 0, byID.Attempts())
	assert.Nil(t, byID.BlockedUntil())

	pending, err := repo.FindPendingByUserID(context.Background(), userA)
	require.NoError(t, err)
	require.NotNil(t, pending)
	assert.Equal(t, requestID1, pending.ID())
}

func TestDeactivationRequestRepository_FindMissingReturnsNil(t *testing.T) {
	pool := pgtest.NewDB(t)
	repo := NewDeactivationRequestRepository(pool)

	byID, err := repo.FindByID(context.Background(), requestID1)
	require.NoError(t, err)
	assert.Nil(t, byID)

	pending, err := repo.FindPendingByUserID(context.Background(), userA)
	require.NoError(t, err)
	assert.Nil(t, pending)
}

func TestDeactivationRequestRepository_Update_PersistsBlockedAfterMaxFailures(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedUser(t, pool, userSpec{id: userA, email: "ana@example.com", nickname: "ana"})
	repo := NewDeactivationRequestRepository(pool)
	req, err := domainUser.NewDeactivationRequest(requestID1, userA, "135790", integrationNow)
	require.NoError(t, err)
	require.NoError(t, repo.Save(context.Background(), req))

	for i := 0; i < domainUser.MaxDeactivationAttempts; i++ {
		req.RegisterFailure(integrationNow.Add(time.Minute))
	}
	require.NoError(t, repo.Update(context.Background(), req))

	got, err := repo.FindByID(context.Background(), requestID1)
	require.NoError(t, err)
	assert.Equal(t, domainUser.MaxDeactivationAttempts, got.Attempts())
	assert.Equal(t, "BLOCKED", got.Status().String())
	require.NotNil(t, got.BlockedUntil())

	stillPending, err := repo.FindPendingByUserID(context.Background(), userA)
	require.NoError(t, err)
	require.NotNil(t, stillPending)
	assert.Equal(t, "BLOCKED", stillPending.Status().String())
}

func TestDeactivationRequestRepository_Update_ConfirmedIsNoLongerPending(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedUser(t, pool, userSpec{id: userA, email: "ana@example.com", nickname: "ana"})
	repo := NewDeactivationRequestRepository(pool)
	req, err := domainUser.NewDeactivationRequest(requestID1, userA, "135790", integrationNow)
	require.NoError(t, err)
	require.NoError(t, repo.Save(context.Background(), req))

	req.Confirm(integrationNow.Add(time.Minute))
	require.NoError(t, repo.Update(context.Background(), req))

	pending, err := repo.FindPendingByUserID(context.Background(), userA)
	require.NoError(t, err)
	assert.Nil(t, pending)
}

func TestDeactivationRequestRepository_InvalidatePendingByUserID_ExpiresPendingAndBlocked(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedUser(t, pool, userSpec{id: userA, email: "ana@example.com", nickname: "ana"})
	repo := NewDeactivationRequestRepository(pool)
	req, err := domainUser.NewDeactivationRequest(requestID1, userA, "135790", integrationNow)
	require.NoError(t, err)
	require.NoError(t, repo.Save(context.Background(), req))

	require.NoError(t, repo.InvalidatePendingByUserID(context.Background(), userA, integrationNow.Add(time.Minute)))

	got, err := repo.FindByID(context.Background(), requestID1)
	require.NoError(t, err)
	assert.Equal(t, "EXPIRED", got.Status().String())
	pending, err := repo.FindPendingByUserID(context.Background(), userA)
	require.NoError(t, err)
	assert.Nil(t, pending)
}

func TestDeactivationAuditLogRepository_Save_StoresTheRecord(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedUser(t, pool, userSpec{id: userA, email: "ana@example.com", nickname: "ana"})
	ip := "10.0.0.1"
	agent := "curl/8"
	entry, err := domainUser.NewDeactivationAuditLog(requestID1, userA, "ana@example.com", "ana", integrationNow, &ip, &agent)
	require.NoError(t, err)

	require.NoError(t, NewDeactivationAuditLogRepository(pool).Save(context.Background(), entry))

	var email, nickname string
	var gotIP, gotAgent *string
	err = pool.QueryRow(context.Background(),
		`SELECT original_email, original_nickname, ip, user_agent FROM deactivation_audit_logs WHERE id = $1`, requestID1,
	).Scan(&email, &nickname, &gotIP, &gotAgent)
	require.NoError(t, err)
	assert.Equal(t, "ana@example.com", email)
	assert.Equal(t, "ana", nickname)
	require.NotNil(t, gotIP)
	assert.Equal(t, "10.0.0.1", *gotIP)
	require.NotNil(t, gotAgent)
	assert.Equal(t, "curl/8", *gotAgent)
}

func TestDeactivationAuditLogRepository_Save_UnknownUserReturnsInternalNotRawError(t *testing.T) {
	pool := pgtest.NewDB(t)
	entry, err := domainUser.NewDeactivationAuditLog(requestID1, userC, "x@example.com", "x", integrationNow, nil, nil)
	require.NoError(t, err)

	err = NewDeactivationAuditLogRepository(pool).Save(context.Background(), entry)

	requireInternal(t, err)
}

func TestGlobalGroupJoiner_AddToGlobalGroup_IsIdempotent(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedUser(t, pool, userSpec{id: userA, email: "ana@example.com", nickname: "ana"})
	joiner := NewGlobalGroupJoiner(pool)

	require.NoError(t, joiner.AddToGlobalGroup(context.Background(), userA))
	require.NoError(t, joiner.AddToGlobalGroup(context.Background(), userA))

	var memberships int
	err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM group_members gm JOIN groups g ON g.id = gm.group_id WHERE g.is_default AND gm.user_id = $1`, userA,
	).Scan(&memberships)
	require.NoError(t, err)
	assert.Equal(t, 1, memberships)
}

func requireInternal(t *testing.T, err error) {
	t.Helper()
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.KindInternal, appErr.Kind)
}

func TestVerificationRepositories_Find_MalformedUUIDReturnsNilNotAnError(t *testing.T) {
	pool := pgtest.NewDB(t)
	ctx := context.Background()
	const bad = "not-a-uuid"

	tests := []struct {
		name string
		find func() (any, error)
	}{
		{"email change by id", func() (any, error) { return NewEmailChangeRepository(pool).FindByID(ctx, bad) }},
		{"email change by code and user", func() (any, error) {
			return NewEmailChangeRepository(pool).FindByCodeAndUserID(ctx, "123456", bad)
		}},
		{"password recovery by id", func() (any, error) { return NewPasswordRecoveryRepository(pool).FindByID(ctx, bad) }},
		{"password recovery pending by user", func() (any, error) {
			return NewPasswordRecoveryRepository(pool).FindPendingByUserID(ctx, bad)
		}},
		{"deactivation request by id", func() (any, error) { return NewDeactivationRequestRepository(pool).FindByID(ctx, bad) }},
		{"deactivation request pending by user", func() (any, error) {
			return NewDeactivationRequestRepository(pool).FindPendingByUserID(ctx, bad)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.find()

			require.NoError(t, err)
			assert.Nil(t, got)
		})
	}
}
