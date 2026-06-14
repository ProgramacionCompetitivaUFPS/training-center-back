package contest

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	appContest "github.com/training-judge-center/backend/internal/application/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// StandingsCache implements application/contest.StandingsCache using Redis.
// Keys: standings:{contestID} (data), lock:standings:{contestID} (refresh lock).
type StandingsCache struct {
	client *redis.Client
}

func NewStandingsCache(client *redis.Client) *StandingsCache {
	return &StandingsCache{client: client}
}

func (c *StandingsCache) Get(ctx context.Context, contestID string) (*appContest.CachedStandings, error) {
	val, err := c.client.Get(ctx, standingsKey(contestID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		slog.ErrorContext(ctx, "standings cache get failed", "contest_id", contestID, "error", err)
		return nil, apperror.NewInternal()
	}

	var cached appContest.CachedStandings
	if err := json.Unmarshal(val, &cached); err != nil {
		slog.ErrorContext(ctx, "standings cache unmarshal failed", "contest_id", contestID, "error", err)
		return nil, apperror.NewInternal()
	}
	return &cached, nil
}

func (c *StandingsCache) Set(ctx context.Context, contestID string, data *appContest.CachedStandings) error {
	b, err := json.Marshal(data)
	if err != nil {
		slog.ErrorContext(ctx, "standings cache marshal failed", "contest_id", contestID, "error", err)
		return apperror.NewInternal()
	}
	if err := c.client.Set(ctx, standingsKey(contestID), b, 0).Err(); err != nil {
		slog.ErrorContext(ctx, "standings cache set failed", "contest_id", contestID, "error", err)
		return apperror.NewInternal()
	}
	return nil
}

func (c *StandingsCache) AcquireRefreshLock(ctx context.Context, contestID string, ttl time.Duration) (bool, error) {
	ok, err := c.client.SetNX(ctx, lockKey(contestID), 1, ttl).Result()
	if err != nil {
		slog.ErrorContext(ctx, "standings refresh lock acquire failed", "contest_id", contestID, "error", err)
		return false, apperror.NewInternal()
	}
	return ok, nil
}

func (c *StandingsCache) ReleaseRefreshLock(ctx context.Context, contestID string) error {
	if err := c.client.Del(ctx, lockKey(contestID)).Err(); err != nil {
		slog.ErrorContext(ctx, "standings refresh lock release failed", "contest_id", contestID, "error", err)
		return apperror.NewInternal()
	}
	return nil
}

func standingsKey(contestID string) string { return "standings:" + contestID }
func lockKey(contestID string) string      { return "lock:standings:" + contestID }
