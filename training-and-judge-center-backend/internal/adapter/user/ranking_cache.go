package user

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type rankingRedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

// RankingCache implements application/user.RankingCache using Redis. Global keys (no
// contestID the way StandingsCache has) — there is one ranking for the whole platform.
type RankingCache struct {
	client rankingRedisClient
}

func NewRankingCache(client *redis.Client) *RankingCache {
	return &RankingCache{client: client}
}

const (
	rankingKey     = "ranking:global"
	rankingLockKey = "lock:ranking:global"
)

func (c *RankingCache) Get(ctx context.Context) (*appuser.CachedRanking, error) {
	val, err := c.client.Get(ctx, rankingKey).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		slog.ErrorContext(ctx, "ranking cache get failed", "error", err)
		return nil, apperror.NewInternal()
	}

	var cached appuser.CachedRanking
	if err := json.Unmarshal(val, &cached); err != nil {
		slog.ErrorContext(ctx, "ranking cache unmarshal failed", "error", err)
		return nil, apperror.NewInternal()
	}
	return &cached, nil
}

func (c *RankingCache) Set(ctx context.Context, data *appuser.CachedRanking) error {
	b, err := json.Marshal(data)
	if err != nil {
		slog.ErrorContext(ctx, "ranking cache marshal failed", "error", err)
		return apperror.NewInternal()
	}
	if err := c.client.Set(ctx, rankingKey, b, 0).Err(); err != nil {
		slog.ErrorContext(ctx, "ranking cache set failed", "error", err)
		return apperror.NewInternal()
	}
	return nil
}

func (c *RankingCache) AcquireRefreshLock(ctx context.Context, ttl time.Duration) (bool, error) {
	ok, err := c.client.SetNX(ctx, rankingLockKey, 1, ttl).Result()
	if err != nil {
		slog.ErrorContext(ctx, "ranking refresh lock acquire failed", "error", err)
		return false, apperror.NewInternal()
	}
	return ok, nil
}

func (c *RankingCache) ReleaseRefreshLock(ctx context.Context) error {
	if err := c.client.Del(ctx, rankingLockKey).Err(); err != nil {
		slog.ErrorContext(ctx, "ranking refresh lock release failed", "error", err)
		return apperror.NewInternal()
	}
	return nil
}

func (c *RankingCache) Invalidate(ctx context.Context) error {
	if err := c.client.Del(ctx, rankingKey, rankingLockKey).Err(); err != nil {
		slog.ErrorContext(ctx, "ranking cache invalidate failed", "error", err)
		return apperror.NewInternal()
	}
	return nil
}
