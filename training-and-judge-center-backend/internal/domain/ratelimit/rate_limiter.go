package ratelimit

import (
	"context"
	"time"
)

type RateLimiter interface {
	Allow(ctx context.Context, key string, maxAttempts int, window time.Duration) (bool, error)
	Reset(ctx context.Context, key string) error
}
