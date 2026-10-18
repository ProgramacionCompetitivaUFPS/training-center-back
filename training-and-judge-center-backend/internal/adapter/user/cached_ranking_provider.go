package user

import (
	"context"
	"log/slog"
	"time"

	appuser "github.com/training-judge-center/backend/internal/application/user"
)

var _ appuser.RankingProvider = (*CachedRankingProvider)(nil)

// CachedRankingProvider implements application/user.RankingProvider by serving the
// platform-wide ranking from RankingCache instead of recomputing it from Postgres on
// every call — same stale-while-revalidate shape as GetStandingsUseCase
// (internal/application/contest/get_standings.go): a cache hit is served immediately
// even if stale, a stale hit kicks a background rebuild for the *next* caller, and a
// cold cache rebuilds synchronously under a refresh lock so concurrent callers don't
// all rebuild at once.
//
// Unlike standings, there is one ranking for the whole platform, not one per contest —
// no contestID, no per-entity cache key.
type CachedRankingProvider struct {
	source    appuser.RankingSource
	cache     appuser.RankingCache
	staleness time.Duration
	lockTTL   time.Duration
}

func NewCachedRankingProvider(source appuser.RankingSource, cache appuser.RankingCache, staleness time.Duration) *CachedRankingProvider {
	return &CachedRankingProvider{
		source:    source,
		cache:     cache,
		staleness: staleness,
		lockTTL:   staleness * 2,
	}
}

func (p *CachedRankingProvider) GetRanking(ctx context.Context, userID string) (int, *int, int, error) {
	cached, err := p.cache.Get(ctx)
	if err != nil {
		return 0, nil, 0, err
	}

	if cached != nil {
		if time.Since(cached.LastUpdated) > p.staleness {
			go p.backgroundRefresh()
		}
		solved, position, totalUsers := rankingOf(cached, userID)
		return solved, position, totalUsers, nil
	}

	rebuilt, err := p.rebuild(ctx)
	if err != nil {
		return 0, nil, 0, err
	}
	solved, position, totalUsers := rankingOf(rebuilt, userID)
	return solved, position, totalUsers, nil
}

func (p *CachedRankingProvider) rebuild(ctx context.Context) (*appuser.CachedRanking, error) {
	solved, totalUsers, err := p.source.GetAllRankings(ctx)
	if err != nil {
		return nil, err
	}
	cached := &appuser.CachedRanking{
		Solved:      solved,
		TotalUsers:  totalUsers,
		LastUpdated: time.Now(),
	}
	if err := p.cache.Set(ctx, cached); err != nil {
		slog.ErrorContext(ctx, "ranking cache set failed after rebuild", "error", err)
	}
	return cached, nil
}

func (p *CachedRankingProvider) backgroundRefresh() {
	ctx := context.Background()
	acquired, err := p.cache.AcquireRefreshLock(ctx, p.lockTTL)
	if err != nil || !acquired {
		return
	}
	defer p.cache.ReleaseRefreshLock(ctx)

	if _, err := p.rebuild(ctx); err != nil {
		slog.ErrorContext(ctx, "ranking background refresh failed", "error", err)
	}
}

// rankingOf reads a user's solved count, 1-based position (nil if they haven't solved
// anything), and the platform's total active users out of a ranking snapshot. Rank is
// 1 + however many users solved strictly more — ties share the same rank, same rule as
// the original single-user query in RankingProvider.GetRanking.
func rankingOf(c *appuser.CachedRanking, userID string) (solved int, position *int, totalUsers int) {
	solved = c.Solved[userID]
	if solved > 0 {
		ahead := 0
		for _, s := range c.Solved {
			if s > solved {
				ahead++
			}
		}
		pos := ahead + 1
		position = &pos
	}
	return solved, position, c.TotalUsers
}
