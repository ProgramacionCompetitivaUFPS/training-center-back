package user_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/user"
	appuser "github.com/training-judge-center/backend/internal/application/user"
)

// fakeRankingSource counts calls and signals each one on a channel, so a test waiting
// on a background refresh can block on that instead of a flaky sleep.
type fakeRankingSource struct {
	mu       sync.Mutex
	calls    int
	called   chan struct{}
	solved   map[string]int
	total    int
}

func newFakeRankingSource(solved map[string]int, total int) *fakeRankingSource {
	return &fakeRankingSource{solved: solved, total: total, called: make(chan struct{}, 10)}
}

func (f *fakeRankingSource) GetAllRankings(context.Context) (map[string]int, int, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	f.called <- struct{}{}
	return f.solved, f.total, nil
}

func (f *fakeRankingSource) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// fakeRankingCache is an in-memory stand-in for appuser.RankingCache — avoids Redis
// serialization noise in tests that only care about the decorator's hit/miss/stale/lock
// logic, not round-tripping through JSON.
type fakeRankingCache struct {
	mu      sync.Mutex
	data    *appuser.CachedRanking
	locked  bool
}

func (c *fakeRankingCache) Get(context.Context) (*appuser.CachedRanking, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.data, nil
}

func (c *fakeRankingCache) Set(_ context.Context, data *appuser.CachedRanking) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = data
	return nil
}

func (c *fakeRankingCache) AcquireRefreshLock(context.Context, time.Duration) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.locked {
		return false, nil
	}
	c.locked = true
	return true, nil
}

func (c *fakeRankingCache) ReleaseRefreshLock(context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.locked = false
	return nil
}

func (c *fakeRankingCache) Invalidate(context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = nil
	c.locked = false
	return nil
}

func TestCachedRankingProvider_ColdCache_RebuildsAndServes(t *testing.T) {
	source := newFakeRankingSource(map[string]int{"u1": 5, "u2": 3}, 10)
	cache := &fakeRankingCache{}
	p := user.NewCachedRankingProvider(source, cache, time.Minute)

	solved, position, totalUsers, err := p.GetRanking(context.Background(), "u1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if solved != 5 {
		t.Errorf("solved = %d, want 5", solved)
	}
	if position == nil || *position != 1 {
		t.Errorf("position = %v, want 1", position)
	}
	if totalUsers != 10 {
		t.Errorf("totalUsers = %d, want 10", totalUsers)
	}
	if source.callCount() != 1 {
		t.Errorf("source called %d times, want exactly 1 (cold rebuild)", source.callCount())
	}
	if cache.data == nil {
		t.Error("expected the rebuild to populate the cache")
	}
}

func TestCachedRankingProvider_FreshHit_DoesNotCallSource(t *testing.T) {
	source := newFakeRankingSource(map[string]int{"u1": 5}, 10)
	cache := &fakeRankingCache{data: &appuser.CachedRanking{
		Solved: map[string]int{"u1": 5}, TotalUsers: 10, LastUpdated: time.Now(),
	}}
	p := user.NewCachedRankingProvider(source, cache, time.Minute)

	solved, _, _, err := p.GetRanking(context.Background(), "u1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if solved != 5 {
		t.Errorf("solved = %d, want 5", solved)
	}
	if source.callCount() != 0 {
		t.Errorf("source called %d times, want 0 (fresh cache hit)", source.callCount())
	}
}

func TestCachedRankingProvider_StaleHit_ServesStaleAndRefreshesInBackground(t *testing.T) {
	source := newFakeRankingSource(map[string]int{"u1": 9}, 20)
	staleData := &appuser.CachedRanking{
		Solved: map[string]int{"u1": 5}, TotalUsers: 10,
		LastUpdated: time.Now().Add(-time.Hour), // long past the 1-minute staleness window
	}
	cache := &fakeRankingCache{data: staleData}
	p := user.NewCachedRankingProvider(source, cache, time.Minute)

	solved, _, totalUsers, err := p.GetRanking(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Served the stale value immediately — never blocks on the rebuild.
	if solved != 5 || totalUsers != 10 {
		t.Errorf("got solved=%d totalUsers=%d, want the stale 5/10 (should not block on refresh)", solved, totalUsers)
	}

	select {
	case <-source.called:
		// background refresh fired, as expected
	case <-time.After(2 * time.Second):
		t.Fatal("expected a background refresh to call the source within 2s")
	}
}

func TestCachedRankingProvider_ZeroSolved_PositionIsNil(t *testing.T) {
	source := newFakeRankingSource(map[string]int{"u1": 5}, 10)
	cache := &fakeRankingCache{}
	p := user.NewCachedRankingProvider(source, cache, time.Minute)

	solved, position, _, err := p.GetRanking(context.Background(), "ghost-user")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if solved != 0 {
		t.Errorf("solved = %d, want 0", solved)
	}
	if position != nil {
		t.Errorf("position = %v, want nil for a user with 0 solved", *position)
	}
}

func TestCachedRankingProvider_TiedUsers_ShareTheSameRank(t *testing.T) {
	source := newFakeRankingSource(map[string]int{"u1": 5, "u2": 5, "u3": 3}, 10)
	cache := &fakeRankingCache{}
	p := user.NewCachedRankingProvider(source, cache, time.Minute)

	_, pos1, _, err := p.GetRanking(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, pos2, _, err := p.GetRanking(context.Background(), "u2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, pos3, _, err := p.GetRanking(context.Background(), "u3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pos1 == nil || pos2 == nil || *pos1 != *pos2 {
		t.Errorf("tied users should share rank: pos1=%v pos2=%v", pos1, pos2)
	}
	if pos1 == nil || *pos1 != 1 {
		t.Errorf("u1/u2 (solved=5, nobody ahead) should be rank 1, got %v", pos1)
	}
	if pos3 == nil || *pos3 != 3 {
		t.Errorf("u3 (solved=3, two users ahead) should be rank 3, got %v", pos3)
	}
}
