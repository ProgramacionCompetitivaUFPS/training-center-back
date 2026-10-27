package user

import (
	"context"
	"time"
)

// RankingProvider is kept separate from ProblemsSolvedProvider so /dashboard
// doesn't pay the cost of this ranking query on every load.
type RankingProvider interface {
	// GetRanking returns:
	//   - problemsSolved: count of unique problems with at least one ACCEPTED submission
	//   - position: rank among all users by problemsSolved (1-based); nil if 0 solved
	//   - totalUsers: count of active users (for ranking context)
	GetRanking(ctx context.Context, userID string) (problemsSolved int, position *int, totalUsers int, err error)
}

// RankingSource computes the platform-wide ranking from scratch — the expensive query
// CachedRankingProvider wraps. Unlike RankingProvider.GetRanking (one user's position),
// this returns every user's solved count in one pass, so a rebuild costs one query
// regardless of how many profile views happen before the cache goes stale again.
type RankingSource interface {
	// GetAllRankings returns every user's count of unique ACCEPTED problems (userID →
	// solved; a user with zero is simply absent) and the count of active users.
	GetAllRankings(ctx context.Context) (solved map[string]int, totalUsers int, err error)
}

// CachedRanking is the full platform ranking snapshot kept in RankingCache — global,
// unlike CachedStandings (one per contest), since there is only one ranking to compute.
type CachedRanking struct {
	Solved      map[string]int // userID -> count of unique ACCEPTED problems; absent means 0
	TotalUsers  int
	LastUpdated time.Time
}

// RankingCache is the global counterpart of StandingsCache (internal/application/contest) —
// same Get/Set/refresh-lock shape, minus the per-contest key, since there is one ranking
// for the whole platform, not one per contest.
type RankingCache interface {
	Get(ctx context.Context) (*CachedRanking, error)
	Set(ctx context.Context, data *CachedRanking) error
	AcquireRefreshLock(ctx context.Context, ttl time.Duration) (bool, error)
	ReleaseRefreshLock(ctx context.Context) error
	Invalidate(ctx context.Context) error
}
