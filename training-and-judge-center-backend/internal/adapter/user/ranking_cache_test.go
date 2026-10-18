package user_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/adapter/user"
)

func newTestRankingCache(t *testing.T) (*user.RankingCache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return user.NewRankingCache(client), mr
}

func TestRankingCache_Get_NoEntry_ReturnsNilWithoutError(t *testing.T) {
	cache, _ := newTestRankingCache(t)
	ctx := context.Background()

	got, err := cache.Get(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for a cache miss, got %+v", got)
	}
}

func TestRankingCache_SetThenGet_RoundTripsData(t *testing.T) {
	cache, _ := newTestRankingCache(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	data := &appuser.CachedRanking{
		Solved:      map[string]int{"u1": 5, "u2": 3},
		TotalUsers:  10,
		LastUpdated: now,
	}

	if err := cache.Set(ctx, data); err != nil {
		t.Fatalf("unexpected error on Set: %v", err)
	}

	got, err := cache.Get(ctx)
	if err != nil {
		t.Fatalf("unexpected error on Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected cached data, got nil")
	}
	if got.Solved["u1"] != 5 || got.Solved["u2"] != 3 {
		t.Fatalf("solved not round-tripped correctly: %+v", got.Solved)
	}
	if got.TotalUsers != 10 {
		t.Fatalf("TotalUsers = %d, want 10", got.TotalUsers)
	}
	if !got.LastUpdated.Equal(now) {
		t.Fatalf("LastUpdated=%v, want %v", got.LastUpdated, now)
	}
}

func TestRankingCache_AcquireRefreshLock_FirstCaller_Succeeds(t *testing.T) {
	cache, _ := newTestRankingCache(t)
	ctx := context.Background()

	acquired, err := cache.AcquireRefreshLock(ctx, time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !acquired {
		t.Fatal("expected the first caller to acquire the lock")
	}
}

func TestRankingCache_AcquireRefreshLock_SecondCaller_FailsWhileHeld(t *testing.T) {
	cache, _ := newTestRankingCache(t)
	ctx := context.Background()

	if _, err := cache.AcquireRefreshLock(ctx, time.Minute); err != nil {
		t.Fatalf("unexpected error on first acquire: %v", err)
	}

	acquired, err := cache.AcquireRefreshLock(ctx, time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if acquired {
		t.Fatal("expected the second caller to be denied while the lock is held")
	}
}

func TestRankingCache_ReleaseRefreshLock_AllowsReacquisition(t *testing.T) {
	cache, _ := newTestRankingCache(t)
	ctx := context.Background()

	if _, err := cache.AcquireRefreshLock(ctx, time.Minute); err != nil {
		t.Fatalf("unexpected error on first acquire: %v", err)
	}
	if err := cache.ReleaseRefreshLock(ctx); err != nil {
		t.Fatalf("unexpected error on release: %v", err)
	}

	acquired, err := cache.AcquireRefreshLock(ctx, time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !acquired {
		t.Fatal("expected the lock to be reacquirable after release")
	}
}

func TestRankingCache_Invalidate_RemovesDataAndLock(t *testing.T) {
	cache, mr := newTestRankingCache(t)
	ctx := context.Background()

	data := &appuser.CachedRanking{Solved: map[string]int{}, LastUpdated: time.Now()}
	if err := cache.Set(ctx, data); err != nil {
		t.Fatalf("unexpected error on Set: %v", err)
	}
	if _, err := cache.AcquireRefreshLock(ctx, time.Minute); err != nil {
		t.Fatalf("unexpected error on acquire: %v", err)
	}

	if err := cache.Invalidate(ctx); err != nil {
		t.Fatalf("unexpected error on Invalidate: %v", err)
	}

	if mr.Exists("ranking:global") {
		t.Fatal("expected the ranking data key to be deleted")
	}
	if mr.Exists("lock:ranking:global") {
		t.Fatal("expected the refresh lock key to be deleted")
	}

	got, err := cache.Get(ctx)
	if err != nil {
		t.Fatalf("unexpected error on Get after invalidate: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil after invalidate, got %+v", got)
	}
}

func TestRankingCache_Get_RedisUnreachable_ReturnsError(t *testing.T) {
	cache, mr := newTestRankingCache(t)
	ctx := context.Background()

	mr.Close()

	_, err := cache.Get(ctx)

	if err == nil {
		t.Fatal("expected an error when Redis is unreachable, got nil")
	}
}

func TestRankingCache_Set_RedisUnreachable_ReturnsError(t *testing.T) {
	cache, mr := newTestRankingCache(t)
	ctx := context.Background()

	mr.Close()

	data := &appuser.CachedRanking{Solved: map[string]int{}, LastUpdated: time.Now()}
	err := cache.Set(ctx, data)

	if err == nil {
		t.Fatal("expected an error when Redis is unreachable, got nil")
	}
}
