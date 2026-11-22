//go:build integration

package contest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/training-judge-center/backend/internal/testutil/pgtest"
)

func TestProblemProvider_FindBySlugs_ResolvesCanAddPerCaller(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	_, err := pool.Exec(context.Background(), `UPDATE problems SET accessibility = 'PRIVATE', author_id = $1 WHERE id = $2`, itUserB, itProblem1)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), `UPDATE problems SET accessibility = 'PRIVATE', status = 'DRAFT', modifiers_ids = ARRAY[$1]::uuid[] WHERE id = $2`, itOwner, itProblem2)
	require.NoError(t, err)
	provider := NewProblemProvider(pool)
	slugs := []string{"two-sum", "three-sum", "missing"}

	tests := []struct {
		name        string
		caller      string
		admin       bool
		wantTwoSum  bool
		wantThreeSm bool
	}{
		{"the author of a private problem", itUserB, false, true, false},
		{"a modifier of a private problem", itOwner, false, false, true},
		{"someone without a link", itUserC, false, false, false},
		{"an admin", itUserC, true, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := provider.FindBySlugs(context.Background(), slugs, tt.caller, tt.admin)

			require.NoError(t, err)
			require.Len(t, got, 2)
			assert.NotContains(t, got, "missing")
			assert.Equal(t, tt.wantTwoSum, got["two-sum"].CanAdd)
			assert.Equal(t, tt.wantThreeSm, got["three-sum"].CanAdd)
			assert.True(t, got["two-sum"].IsPublished)
			assert.False(t, got["three-sum"].IsPublished)
		})
	}

	empty, err := provider.FindBySlugs(context.Background(), nil, itOwner, false)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestProblemProvider_FindBySlugs_PublicProblemCanBeAddedByAnyone(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)

	got, err := NewProblemProvider(pool).FindBySlugs(context.Background(), []string{"two-sum"}, itUserC, false)

	require.NoError(t, err)
	assert.True(t, got["two-sum"].CanAdd)
}

func TestProblemProvider_FindByIDs_ReturnsOnlyExisting(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	provider := NewProblemProvider(pool)

	got, err := provider.FindByIDs(context.Background(), []string{itProblem1, itProblem2, itMissing})

	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "two-sum", got[itProblem1].Slug)
	assert.Equal(t, "three-sum", got[itProblem2].Title)

	empty, err := provider.FindByIDs(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestProblemProvider_FindByIDsWithLimits_ReadsTimeAndMemoryLimits(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	_, err := pool.Exec(context.Background(), `UPDATE problems SET time_limit = 2000, memory_limit = 256 WHERE id = $1`, itProblem1)
	require.NoError(t, err)
	provider := NewProblemProvider(pool)

	got, err := provider.FindByIDsWithLimits(context.Background(), []string{itProblem1})

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, 2000, got[itProblem1].TimeLimit)
	assert.Equal(t, 256, got[itProblem1].MemoryLimit)

	empty, err := provider.FindByIDsWithLimits(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestProblemProvider_FindByIDsWithLimits_ProblemWithoutLimitsIsAnInternalError(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)

	_, err := NewProblemProvider(pool).FindByIDsWithLimits(context.Background(), []string{itProblem1})

	require.Error(t, err)
}

func TestGroupProvider_FindByIDAndFindByIDs(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	_, err := pool.Exec(context.Background(), `UPDATE groups SET visibility = 'NOT_VISIBLE' WHERE id = $1`, itGroupB)
	require.NoError(t, err)
	provider := NewGroupProvider(pool)

	visible, err := provider.FindByID(context.Background(), itGroupA)
	require.NoError(t, err)
	require.NotNil(t, visible)
	assert.Equal(t, "Group A", visible.Name)
	assert.True(t, visible.IsVisible)

	hidden, err := provider.FindByID(context.Background(), itGroupB)
	require.NoError(t, err)
	assert.False(t, hidden.IsVisible)

	missing, err := provider.FindByID(context.Background(), itMissing)
	require.NoError(t, err)
	assert.Nil(t, missing)

	malformed, err := provider.FindByID(context.Background(), "not-a-uuid")
	require.NoError(t, err)
	assert.Nil(t, malformed)

	many, err := provider.FindByIDs(context.Background(), []string{itGroupA, itGroupB, itMissing})
	require.NoError(t, err)
	require.Len(t, many, 2)
	assert.False(t, many[itGroupB].IsVisible)

	empty, err := provider.FindByIDs(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestGroupProvider_ListAccessibleGroupIDs(t *testing.T) {
	pool := pgtest.NewDB(t)
	itSeedBase(t, pool)
	_, err := pool.Exec(context.Background(), `UPDATE groups SET visibility = 'NOT_VISIBLE' WHERE id = $1`, itGroupB)
	require.NoError(t, err)
	pgtest.SeedGroupMember(t, pool, "77770000-0000-0000-0000-0000000000a1", itGroupB, itUserB, "MEMBER")
	provider := NewGroupProvider(pool)

	admin, err := provider.ListAccessibleGroupIDs(context.Background(), itUserC, true)
	require.NoError(t, err)
	assert.Nil(t, admin, "nil means every group")

	outsider, err := provider.ListAccessibleGroupIDs(context.Background(), itUserC, false)
	require.NoError(t, err)
	assert.Contains(t, outsider, itGroupA)
	assert.NotContains(t, outsider, itGroupB)

	member, err := provider.ListAccessibleGroupIDs(context.Background(), itUserB, false)
	require.NoError(t, err)
	assert.Contains(t, member, itGroupA)
	assert.Contains(t, member, itGroupB)
}
