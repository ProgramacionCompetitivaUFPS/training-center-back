package ratelimit_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/training-judge-center/backend/internal/adapter/ratelimit"
)

func newTestLimiter(t *testing.T) (*ratelimit.RedisRateLimiter, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return ratelimit.NewRedisRateLimiter(client), mr
}

func TestAllow_BelowLimit_ReturnsTrue(t *testing.T) {
	// Arrange
	limiter, _ := newTestLimiter(t)
	ctx := context.Background()

	// Act
	allowed, err := limiter.Allow(ctx, "test:key", 3, time.Minute)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("expected request to be allowed, got denied")
	}
}

func TestAllow_AtLimit_ReturnsTrue(t *testing.T) {
	// Arrange
	limiter, _ := newTestLimiter(t)
	ctx := context.Background()
	key := "test:key"
	maxAttempts := 3

	// Act — make exactly maxAttempts calls
	var allowed bool
	var err error
	for i := 0; i < maxAttempts; i++ {
		allowed, err = limiter.Allow(ctx, key, maxAttempts, time.Minute)
		if err != nil {
			t.Fatalf("unexpected error on attempt %d: %v", i+1, err)
		}
	}

	// Assert — the attempt at count == maxAttempts should still be allowed
	if !allowed {
		t.Fatal("expected request exactly at the limit to be allowed, got denied")
	}
}

func TestAllow_ExceedsLimit_ReturnsFalse(t *testing.T) {
	// Arrange
	limiter, _ := newTestLimiter(t)
	ctx := context.Background()
	key := "test:key"
	maxAttempts := 3

	for i := 0; i < maxAttempts; i++ {
		_, _ = limiter.Allow(ctx, key, maxAttempts, time.Minute)
	}

	// Act — one more attempt beyond the limit
	allowed, err := limiter.Allow(ctx, key, maxAttempts, time.Minute)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Fatal("expected request to be denied after exceeding limit, got allowed")
	}
}

func TestAllow_FirstCall_SetsExpiry(t *testing.T) {
	// Arrange
	limiter, mr := newTestLimiter(t)
	ctx := context.Background()
	key := "test:key"
	window := 2 * time.Minute

	// Act
	_, err := limiter.Allow(ctx, key, 5, window)

	// Assert — verifies the Lua script called EXPIRE correctly on the first increment
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ttl := mr.TTL(key)
	if ttl <= 0 {
		t.Fatalf("expected key to have a positive TTL after first call, got %v", ttl)
	}
}

func TestAllow_SubsequentCalls_DoNotResetExpiry(t *testing.T) {
	// Arrange
	limiter, mr := newTestLimiter(t)
	ctx := context.Background()
	key := "test:key"
	window := 2 * time.Minute

	_, _ = limiter.Allow(ctx, key, 5, window)
	ttlAfterFirst := mr.TTL(key)

	// Advance simulated time so the TTL decreases
	mr.FastForward(30 * time.Second)

	// Act — second call
	_, err := limiter.Allow(ctx, key, 5, window)

	// Assert — the TTL must not have been reset back to the full window duration
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ttlAfterSecond := mr.TTL(key)
	if ttlAfterSecond >= ttlAfterFirst {
		t.Fatal("expected second call not to reset the TTL, but TTL was not reduced")
	}
}

func TestAllow_ScriptError_ReturnsError(t *testing.T) {
	// Arrange
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	limiter := ratelimit.NewRedisRateLimiter(client)
	ctx := context.Background()

	mr.Close() // simulate Redis being unreachable

	// Act
	_, err := limiter.Allow(ctx, "test:key", 5, time.Minute)

	// Assert
	if err == nil {
		t.Fatal("expected error when Redis is unreachable, got nil")
	}
}

func TestReset_DeletesKey(t *testing.T) {
	// Arrange
	limiter, mr := newTestLimiter(t)
	ctx := context.Background()
	key := "test:key"

	_, _ = limiter.Allow(ctx, key, 5, time.Minute)
	if !mr.Exists(key) {
		t.Fatal("precondition failed: expected key to exist before reset")
	}

	// Act
	err := limiter.Reset(ctx, key)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error on Reset: %v", err)
	}
	if mr.Exists(key) {
		t.Fatal("expected key to be deleted after reset, but it still exists")
	}
}

func TestAllow_AfterReset_StartsFromOne(t *testing.T) {
	// Arrange
	limiter, _ := newTestLimiter(t)
	ctx := context.Background()
	key := "test:key"
	maxAttempts := 2

	for i := 0; i <= maxAttempts; i++ {
		_, _ = limiter.Allow(ctx, key, maxAttempts, time.Minute)
	}

	// Act — reset and make one fresh request in the new window
	_ = limiter.Reset(ctx, key)
	allowed, err := limiter.Allow(ctx, key, maxAttempts, time.Minute)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error after reset: %v", err)
	}
	if !allowed {
		t.Fatal("expected request to be allowed after reset, got denied")
	}
}
