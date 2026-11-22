//go:build integration

package problem

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/training-judge-center/backend/internal/testutil/pgtest"
)

func insertUser(t *testing.T, pool *pgxpool.Pool, id, nickname, name string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO users (id, email, password, name, nickname, country, city, institution)
		VALUES ($1, $2, 'hash', $3, $4, 'Colombia', 'Bogota', 'UNAL')`,
		id, nickname+"@example.com", name, nickname)
	require.NoError(t, err)
}

func TestUserProvider_ExistsByID(t *testing.T) {
	pool := pgtest.NewDB(t)
	insertUser(t, pool, authorA, "ana", "Ana Perez")
	provider := NewUserProvider(pool)

	exists, err := provider.ExistsByID(context.Background(), authorA)
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = provider.ExistsByID(context.Background(), authorB)
	require.NoError(t, err)
	assert.False(t, exists)

	exists, err = provider.ExistsByID(context.Background(), "not-a-uuid")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestUserProvider_GetDisplay_FoundAndUnknown(t *testing.T) {
	pool := pgtest.NewDB(t)
	insertUser(t, pool, authorA, "ana", "Ana Perez")
	provider := NewUserProvider(pool)

	found, err := provider.GetDisplay(context.Background(), authorA)
	require.NoError(t, err)
	assert.Equal(t, "ana", found.Nickname)
	assert.Equal(t, "Ana Perez", found.Name)

	missing, err := provider.GetDisplay(context.Background(), authorB)
	require.NoError(t, err)
	assert.Equal(t, "unknown", missing.Nickname)
	assert.Equal(t, "", missing.Name)

	malformed, err := provider.GetDisplay(context.Background(), "not-a-uuid")
	require.NoError(t, err)
	assert.Equal(t, "unknown", malformed.Nickname)
}

func TestUserProvider_GetDisplays_ReturnsOnlyExistingUsers(t *testing.T) {
	pool := pgtest.NewDB(t)
	insertUser(t, pool, authorA, "ana", "Ana Perez")
	insertUser(t, pool, authorB, "beto", "Beto Ruiz")
	provider := NewUserProvider(pool)

	got, err := provider.GetDisplays(context.Background(), []string{authorA, authorB, viewerC})

	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "ana", got[authorA].Nickname)
	assert.Equal(t, "Beto Ruiz", got[authorB].Name)
	assert.NotContains(t, got, viewerC)
}

func TestUserProvider_GetDisplays_EmptyInputReturnsEmptyMap(t *testing.T) {
	pool := pgtest.NewDB(t)

	got, err := NewUserProvider(pool).GetDisplays(context.Background(), nil)

	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestUserProvider_GetIDByNickname_IsCaseInsensitive(t *testing.T) {
	pool := pgtest.NewDB(t)
	insertUser(t, pool, authorA, "ana", "Ana Perez")
	provider := NewUserProvider(pool)

	id, found, err := provider.GetIDByNickname(context.Background(), "ANA")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, authorA, id)

	_, found, err = provider.GetIDByNickname(context.Background(), "nadie")
	require.NoError(t, err)
	assert.False(t, found)
}
