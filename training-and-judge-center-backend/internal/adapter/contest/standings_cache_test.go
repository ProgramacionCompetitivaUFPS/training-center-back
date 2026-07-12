package contest_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/training-judge-center/backend/internal/adapter/contest"
	appContest "github.com/training-judge-center/backend/internal/application/contest"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
)

func newTestStandingsCache(t *testing.T) (*contest.StandingsCache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return contest.NewStandingsCache(client), mr
}

func TestGet_NoEntry_ReturnsNilWithoutError(t *testing.T) {
	cache, _ := newTestStandingsCache(t)
	ctx := context.Background()

	got, err := cache.Get(ctx, "contest-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for a cache miss, got %+v", got)
	}
}

func TestSetThenGet_RoundTripsData(t *testing.T) {
	cache, _ := newTestStandingsCache(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	data := &appContest.CachedStandings{
		Participants: []domainContest.ParticipantStanding{
			{ContestantID: "u1", ParticipantType: "INDIVIDUAL", Problems: map[string]domainContest.ProblemAttempt{}},
		},
		TeamMembers: map[string][]string{"team-1": {"m1", "m2"}},
		Profiles: map[string]*appContest.ParticipantProfile{
			"u1": {ID: "u1", Country: "colombia", City: "bogota", Institution: "ufps"},
		},
		LastUpdated: now,
	}

	if err := cache.Set(ctx, "contest-1", data); err != nil {
		t.Fatalf("unexpected error on Set: %v", err)
	}

	got, err := cache.Get(ctx, "contest-1")
	if err != nil {
		t.Fatalf("unexpected error on Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected cached data, got nil")
	}
	if len(got.Participants) != 1 || got.Participants[0].ContestantID != "u1" {
		t.Fatalf("participants not round-tripped correctly: %+v", got.Participants)
	}
	if len(got.TeamMembers["team-1"]) != 2 {
		t.Fatalf("team members not round-tripped correctly: %+v", got.TeamMembers)
	}
	if got.Profiles["u1"] == nil || got.Profiles["u1"].Country != "colombia" {
		t.Fatalf("profiles not round-tripped correctly: %+v", got.Profiles)
	}
	if !got.LastUpdated.Equal(now) {
		t.Fatalf("LastUpdated=%v, want %v", got.LastUpdated, now)
	}
}

func TestAcquireRefreshLock_FirstCaller_Succeeds(t *testing.T) {
	cache, _ := newTestStandingsCache(t)
	ctx := context.Background()

	acquired, err := cache.AcquireRefreshLock(ctx, "contest-1", time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !acquired {
		t.Fatal("expected the first caller to acquire the lock")
	}
}

func TestAcquireRefreshLock_SecondCaller_FailsWhileHeld(t *testing.T) {
	cache, _ := newTestStandingsCache(t)
	ctx := context.Background()

	if _, err := cache.AcquireRefreshLock(ctx, "contest-1", time.Minute); err != nil {
		t.Fatalf("unexpected error on first acquire: %v", err)
	}

	acquired, err := cache.AcquireRefreshLock(ctx, "contest-1", time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if acquired {
		t.Fatal("expected the second caller to be denied while the lock is held")
	}
}

func TestReleaseRefreshLock_AllowsReacquisition(t *testing.T) {
	cache, _ := newTestStandingsCache(t)
	ctx := context.Background()

	if _, err := cache.AcquireRefreshLock(ctx, "contest-1", time.Minute); err != nil {
		t.Fatalf("unexpected error on first acquire: %v", err)
	}
	if err := cache.ReleaseRefreshLock(ctx, "contest-1"); err != nil {
		t.Fatalf("unexpected error on release: %v", err)
	}

	acquired, err := cache.AcquireRefreshLock(ctx, "contest-1", time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !acquired {
		t.Fatal("expected the lock to be reacquirable after release")
	}
}

func TestInvalidate_RemovesDataAndLock(t *testing.T) {
	cache, mr := newTestStandingsCache(t)
	ctx := context.Background()

	data := &appContest.CachedStandings{Participants: []domainContest.ParticipantStanding{}, LastUpdated: time.Now()}
	if err := cache.Set(ctx, "contest-1", data); err != nil {
		t.Fatalf("unexpected error on Set: %v", err)
	}
	if _, err := cache.AcquireRefreshLock(ctx, "contest-1", time.Minute); err != nil {
		t.Fatalf("unexpected error on acquire: %v", err)
	}

	if err := cache.Invalidate(ctx, "contest-1"); err != nil {
		t.Fatalf("unexpected error on Invalidate: %v", err)
	}

	if mr.Exists("standings:contest-1") {
		t.Fatal("expected the standings data key to be deleted")
	}
	if mr.Exists("lock:standings:contest-1") {
		t.Fatal("expected the refresh lock key to be deleted")
	}

	got, err := cache.Get(ctx, "contest-1")
	if err != nil {
		t.Fatalf("unexpected error on Get after invalidate: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil after invalidate, got %+v", got)
	}
}

func TestGet_RedisUnreachable_ReturnsError(t *testing.T) {
	cache, mr := newTestStandingsCache(t)
	ctx := context.Background()

	mr.Close() // simulate Redis being unreachable

	_, err := cache.Get(ctx, "contest-1")

	if err == nil {
		t.Fatal("expected an error when Redis is unreachable, got nil")
	}
}

func TestSet_RedisUnreachable_ReturnsError(t *testing.T) {
	cache, mr := newTestStandingsCache(t)
	ctx := context.Background()

	mr.Close() // simulate Redis being unreachable

	data := &appContest.CachedStandings{Participants: []domainContest.ParticipantStanding{}, LastUpdated: time.Now()}
	err := cache.Set(ctx, "contest-1", data)

	if err == nil {
		t.Fatal("expected an error when Redis is unreachable, got nil")
	}
}
