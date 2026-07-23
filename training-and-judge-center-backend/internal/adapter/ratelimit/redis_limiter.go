package ratelimit

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var rateLimitScript = redis.NewScript(`
	local count = redis.call('INCR', KEYS[1])
	if count == 1 then
		redis.call('EXPIRE', KEYS[1], ARGV[1])
	end
	return count
`)

type RedisRateLimiter struct {
	client *redis.Client
}

func NewRedisRateLimiter(client *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{client: client}
}

func (r *RedisRateLimiter) Allow(ctx context.Context, key string, maxAttempts int, window time.Duration) (bool, error) {
	count, err := rateLimitScript.Run(ctx, r.client, []string{key}, int64(window.Seconds())).Int64()
	if err != nil {
		slog.ErrorContext(ctx, "rate limit script failed", "key", key, "error", err)
		return false, apperror.NewInternal()
	}
	return count <= int64(maxAttempts), nil
}

func (r *RedisRateLimiter) Reset(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		slog.ErrorContext(ctx, "failed to reset rate limit key", "key", key, "error", err)
		return apperror.NewInternal()
	}
	return nil
}
