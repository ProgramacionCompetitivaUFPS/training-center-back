package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/training-judge-center/backend/internal/adapter/auth"
	appuser "github.com/training-judge-center/backend/internal/application/user"
)

func newTestCache(t *testing.T) (*auth.RedisRotationCache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return auth.NewRedisRotationCache(client), mr
}

func TestSave_Get_RoundTrips(t *testing.T) {
	// Arrange
	cache, _ := newTestCache(t)
	ctx := context.Background()
	output := appuser.RefreshOutput{
		Token:            "access-token",
		RefreshToken:     "refresh-secret",
		SessionExpiresAt: time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC),
	}

	// Act
	if err := cache.Save(ctx, "old-hash", output, time.Minute); err != nil {
		t.Fatalf("unexpected error on Save: %v", err)
	}
	got, err := cache.Get(ctx, "old-hash")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error on Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected a cached output, got nil")
	}
	if got.Token != output.Token || got.RefreshToken != output.RefreshToken || !got.SessionExpiresAt.Equal(output.SessionExpiresAt) {
		t.Errorf("expected %+v, got %+v", output, *got)
	}
}

func TestGet_UnknownHash_ReturnsNilNil(t *testing.T) {
	// Arrange
	cache, _ := newTestCache(t)
	ctx := context.Background()

	// Act
	got, err := cache.Get(ctx, "never-saved")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for an unknown hash, got %+v", got)
	}
}

func TestSave_SetsExactTTL(t *testing.T) {
	// Arrange
	cache, mr := newTestCache(t)
	ctx := context.Background()
	ttl := 10 * time.Second

	// Act
	if err := cache.Save(ctx, "old-hash", appuser.RefreshOutput{}, ttl); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Assert
	got := mr.TTL("rotation:old-hash")
	if got != ttl {
		t.Errorf("expected TTL %v, got %v", ttl, got)
	}
}

func TestGet_AfterTTLExpires_ReturnsNilNil(t *testing.T) {
	// Arrange
	cache, mr := newTestCache(t)
	ctx := context.Background()
	ttl := 10 * time.Second

	if err := cache.Save(ctx, "old-hash", appuser.RefreshOutput{}, ttl); err != nil {
		t.Fatalf("unexpected error on Save: %v", err)
	}

	// Act — advance simulated time past the TTL
	mr.FastForward(ttl + time.Second)
	got, err := cache.Get(ctx, "old-hash")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil after TTL expiry, got %+v", got)
	}
}

func TestSave_RedisUnreachable_ReturnsError(t *testing.T) {
	// Arrange
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := auth.NewRedisRotationCache(client)
	ctx := context.Background()
	mr.Close()

	// Act
	err := cache.Save(ctx, "old-hash", appuser.RefreshOutput{}, time.Minute)

	// Assert
	if err == nil {
		t.Fatal("expected error when Redis is unreachable, got nil")
	}
}

func TestGet_RedisUnreachable_ReturnsError(t *testing.T) {
	// Arrange
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := auth.NewRedisRotationCache(client)
	ctx := context.Background()
	mr.Close()

	// Act
	_, err := cache.Get(ctx, "old-hash")

	// Assert
	if err == nil {
		t.Fatal("expected error when Redis is unreachable, got nil")
	}
}
