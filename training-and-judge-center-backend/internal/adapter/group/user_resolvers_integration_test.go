//go:build integration

package group

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/training-judge-center/backend/internal/testutil/pgtest"
)

func TestEmailResolver_ResolveByEmail_IsCaseInsensitiveOnTheInput(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedUsers(t, pool)
	resolver := NewEmailResolver(pool)

	got, err := resolver.ResolveByEmail(context.Background(), "ANA@example.com")

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, itUserA, got.ID)
	assert.Equal(t, "ana", got.Nickname)
	assert.Equal(t, "ana@example.com", got.Email)
	assert.Equal(t, "CONTESTANT", got.SystemRole)

	missing, err := resolver.ResolveByEmail(context.Background(), "nobody@example.com")
	require.NoError(t, err)
	assert.Nil(t, missing)
}

func TestNicknameResolver_ResolveByNickname_FindsDeactivatedAccountsWithoutEmail(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedUsers(t, pool)
	_, err := pool.Exec(context.Background(), `UPDATE users SET status = 'DEACTIVATED', email = NULL, deactivated_at = NOW() WHERE id = $1`, itUserB)
	require.NoError(t, err)
	resolver := NewNicknameResolver(pool)

	active, err := resolver.ResolveByNickname(context.Background(), "ANA")
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, itUserA, active.ID)

	deactivated, err := resolver.ResolveByNickname(context.Background(), "beto")
	require.NoError(t, err)
	require.NotNil(t, deactivated)
	assert.Equal(t, "", deactivated.Email)

	missing, err := resolver.ResolveByNickname(context.Background(), "nobody")
	require.NoError(t, err)
	assert.Nil(t, missing)
}

func TestPreferencesReader_HideGlobalGroup(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedUsers(t, pool)
	_, err := pool.Exec(context.Background(), `UPDATE users SET preferences = '{"hideGlobalGroup": true}'::jsonb WHERE id = $1`, itUserA)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), `UPDATE users SET preferences = '{"hideGlobalGroup": "not-a-bool"}'::jsonb WHERE id = $1`, itUserB)
	require.NoError(t, err)
	reader := NewPreferencesReader(pool)

	hidden, err := reader.HideGlobalGroup(context.Background(), itUserA)
	require.NoError(t, err)
	assert.True(t, hidden)

	unset, err := reader.HideGlobalGroup(context.Background(), itUserC)
	require.NoError(t, err)
	assert.False(t, unset)

	corrupt, err := reader.HideGlobalGroup(context.Background(), itUserB)
	require.NoError(t, err)
	assert.False(t, corrupt, "an unreadable preference falls back to showing the group")

	unknown, err := reader.HideGlobalGroup(context.Background(), itMissing)
	require.NoError(t, err)
	assert.False(t, unknown)
}

func TestUserProvider_GetDisplays_IncludesDeactivatedAccounts(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedUsers(t, pool)
	_, err := pool.Exec(context.Background(), `UPDATE users SET status = 'DEACTIVATED', email = NULL, deactivated_at = NOW() WHERE id = $1`, itUserB)
	require.NoError(t, err)
	provider := NewUserProvider(pool)

	got, err := provider.GetDisplays(context.Background(), []string{itUserA, itUserB, itMissing})

	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "ana@example.com", got[itUserA].Email)
	assert.Equal(t, "", got[itUserB].Email)

	empty, err := provider.GetDisplays(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}
