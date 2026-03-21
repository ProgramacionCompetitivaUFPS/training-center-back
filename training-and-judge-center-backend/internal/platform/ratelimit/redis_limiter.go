package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisRateLimiter struct {
	client *redis.Client
}

func NewRedisRateLimiter(client *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{client: client}
}

func (r *RedisRateLimiter) Allow(ctx context.Context, key string, maxAttempts int, window time.Duration) (bool, error) {
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to increment rate limit: %w", err)
	}

	if count == 1 {
		r.client.Expire(ctx, key, window)
	}

	if count > int64(maxAttempts) {
		return false, nil
	}

	return true, nil
}

func (r *RedisRateLimiter) Reset(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}
