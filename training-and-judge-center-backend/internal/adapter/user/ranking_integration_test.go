//go:build integration

package user

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/training-judge-center/backend/internal/testutil/pgtest"
)

const (
	rankProblem1 = "aaaaaaaa-0000-0000-0000-000000000001"
	rankProblem2 = "aaaaaaaa-0000-0000-0000-000000000002"
)

func seedRanking(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	seedDirectory(t, pool)
	pgtest.SeedProblem(t, pool, rankProblem1, "two-sum", "PUBLISHED")
	pgtest.SeedProblem(t, pool, rankProblem2, "three-sum", "PUBLISHED")
	sub := func(id, problem, user, status string) {
		pgtest.SeedSubmission(t, pool, pgtest.Submission{ID: id, ProblemID: problem, UserID: user, Status: status})
	}
	sub("a1", rankProblem1, userA, "ACCEPTED")
	sub("a1-again", rankProblem1, userA, "ACCEPTED")
	sub("a2", rankProblem2, userA, "ACCEPTED")
	sub("b1", rankProblem1, userB, "ACCEPTED")
	sub("b2", rankProblem2, userB, "WRONG_ANSWER")
	sub("c1", rankProblem1, userC, "WRONG_ANSWER")
}

func TestRankingProvider_GetRanking_PositionCountsOnlyUsersWithStrictlyMoreSolved(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedRanking(t, pool)
	provider := NewRankingProvider(pool)

	tests := []struct {
		name         string
		user         string
		wantSolved   int
		wantPosition *int
	}{
		{"most problems solved", userA, 2, intPtr(1)},
		{"one problem solved", userB, 1, intPtr(2)},
		{"nothing solved has no position", userC, 0, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			solved, position, total, err := provider.GetRanking(context.Background(), tt.user)

			require.NoError(t, err)
			assert.Equal(t, tt.wantSolved, solved)
			assert.Equal(t, tt.wantPosition, position)
			assert.Equal(t, 3, total, "only active users count, the seeded system user does not")
		})
	}
}

func TestRankingProvider_GetAllRankings_ReturnsEverySolverAndTheActiveTotal(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedRanking(t, pool)

	solved, total, err := NewRankingProvider(pool).GetAllRankings(context.Background())

	require.NoError(t, err)
	assert.Equal(t, map[string]int{userA: 2, userB: 1}, solved)
	assert.Equal(t, 3, total)
}

func TestProblemsSolvedProvider_GetProblemsSolved_CountsDistinctAcceptedProblems(t *testing.T) {
	pool := pgtest.NewDB(t)
	seedRanking(t, pool)
	provider := NewProblemsSolvedProvider(pool)

	two, err := provider.GetProblemsSolved(context.Background(), userA)
	require.NoError(t, err)
	assert.Equal(t, 2, two)

	none, err := provider.GetProblemsSolved(context.Background(), userC)
	require.NoError(t, err)
	assert.Equal(t, 0, none)
}

func intPtr(n int) *int { return &n }
