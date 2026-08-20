package pool

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/moby/moby/client"
)

// smallMemBytes is the per-container memory used in all tests (64 MB).
const smallMemBytes = 64 * 1024 * 1024

// testCfg returns a PoolConfig that fits exactly `capacity` containers of 64 MB each.
// IdleTimeout and ReapInterval are set to 1 hour so the reaper never fires during
// non-reaper tests.
func testCfg(capacity int) PoolConfig {
	return PoolConfig{
		BudgetBytes:  int64(capacity) * smallMemBytes,
		IdleTimeout:  time.Hour,
		ReapInterval: time.Hour,
		Languages: map[string]LanguageConfig{
			"cpp20":  {Image: "judge:cpp20", MemoryBytes: smallMemBytes},
			"java17": {Image: "judge:java17", MemoryBytes: smallMemBytes},
		},
	}
}

// mockDockerClient records calls and delegates to optional fn fields.
// Default behaviour: ContainerCreate succeeds with an auto-incremented ID;
// ContainerStart and ContainerRemove succeed with empty results.
type mockDockerClient struct {
	createFn  func(context.Context, client.ContainerCreateOptions) (client.ContainerCreateResult, error)
	startFn   func(context.Context, string, client.ContainerStartOptions) (client.ContainerStartResult, error)
	removeFn  func(context.Context, string, client.ContainerRemoveOptions) (client.ContainerRemoveResult, error)
	pingFn    func(context.Context, client.PingOptions) (client.PingResult, error)
	createCnt atomic.Int64
	removeCnt atomic.Int64
	idCounter atomic.Int64
}

func (m *mockDockerClient) ContainerCreate(ctx context.Context, opts client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
	m.createCnt.Add(1)
	if m.createFn != nil {
		return m.createFn(ctx, opts)
	}
	return client.ContainerCreateResult{ID: fmt.Sprintf("ctr-%d", m.idCounter.Add(1))}, nil
}

func (m *mockDockerClient) ContainerStart(ctx context.Context, id string, opts client.ContainerStartOptions) (client.ContainerStartResult, error) {
	if m.startFn != nil {
		return m.startFn(ctx, id, opts)
	}
	return client.ContainerStartResult{}, nil
}

func (m *mockDockerClient) ContainerRemove(ctx context.Context, id string, opts client.ContainerRemoveOptions) (client.ContainerRemoveResult, error) {
	m.removeCnt.Add(1)
	if m.removeFn != nil {
		return m.removeFn(ctx, id, opts)
	}
	return client.ContainerRemoveResult{}, nil
}

func (m *mockDockerClient) Ping(ctx context.Context, opts client.PingOptions) (client.PingResult, error) {
	if m.pingFn != nil {
		return m.pingFn(ctx, opts)
	}
	return client.PingResult{}, nil
}

func TestIsHealthy_ReturnsTrueWhenDaemonResponds(t *testing.T) {
	p := newTestPool(t, testCfg(1), &mockDockerClient{})
	if !p.IsHealthy(context.Background()) {
		t.Error("expected healthy pool when daemon responds")
	}
}

func TestIsHealthy_ReturnsFalseWhenDaemonUnreachable(t *testing.T) {
	docker := &mockDockerClient{
		pingFn: func(context.Context, client.PingOptions) (client.PingResult, error) {
			return client.PingResult{}, errors.New("daemon unreachable")
		},
	}
	p := newTestPool(t, testCfg(1), docker)
	if p.IsHealthy(context.Background()) {
		t.Error("expected unhealthy pool when daemon is unreachable")
	}
}

// newTestPool creates a started Pool and registers p.Stop as a test cleanup,
// which guarantees the reaper goroutine is joined after every test.
func newTestPool(t *testing.T, cfg PoolConfig, docker dockerLifecycle) *Pool {
	t.Helper()
	p := NewPool(cfg, docker)
	p.Start()
	t.Cleanup(p.Stop)
	return p
}

// injectedIDCounter gives unique IDs to containers created by putIdleContainer.
var injectedIDCounter atomic.Int64

// putIdleContainer injects a pre-built idle container directly into the pool,
// bypassing Docker. Used to set up test preconditions.
func putIdleContainer(t *testing.T, p *Pool, lang string, age time.Duration) *Container {
	t.Helper()
	c := &Container{
		id:          fmt.Sprintf("ctr-injected-%d", injectedIDCounter.Add(1)),
		language:    lang,
		memoryBytes: smallMemBytes,
		state:       stateIdle,
		lastUsedAt:  time.Now().Add(-age),
	}
	p.mu.Lock()
	p.containers = append(p.containers, c)
	p.allocatedBytes += c.memoryBytes
	p.mu.Unlock()
	return c
}

func assertAllocatedBytes(t *testing.T, p *Pool, want int64) {
	t.Helper()
	p.mu.Lock()
	got := p.allocatedBytes
	p.mu.Unlock()
	if got != want {
		t.Errorf("allocatedBytes = %d, want %d", got, want)
	}
}

func assertState(t *testing.T, p *Pool, c *Container, want containerState) {
	t.Helper()
	p.mu.Lock()
	got := c.state
	p.mu.Unlock()
	if got != want {
		t.Errorf("container state = %v, want %v", got, want)
	}
}

// --- Tests ---

// 1. Fast path: an idle container of the requested language is returned directly
// without creating a new one.
func TestClaim_IdleContainerExists_ReturnedWithoutCreating(t *testing.T) {
	mock := &mockDockerClient{}
	p := newTestPool(t, testCfg(2), mock)
	c := putIdleContainer(t, p, "cpp20", 0)

	got, err := p.Claim(context.Background(), "cpp20")
	if err != nil {
		t.Fatal(err)
	}
	if got != c {
		t.Error("expected existing container, got a different one")
	}
	if mock.createCnt.Load() != 0 {
		t.Errorf("createCnt = %d, want 0", mock.createCnt.Load())
	}
	assertAllocatedBytes(t, p, smallMemBytes)
	assertState(t, p, got, stateBusy)
}

// 2. Empty pool: Claim creates a new container and updates allocatedBytes.
func TestClaim_NoPreviousContainers_CreatesNew(t *testing.T) {
	mock := &mockDockerClient{}
	p := newTestPool(t, testCfg(1), mock)

	got, err := p.Claim(context.Background(), "cpp20")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected container, got nil")
	}
	if mock.createCnt.Load() != 1 {
		t.Errorf("createCnt = %d, want 1", mock.createCnt.Load())
	}
	assertAllocatedBytes(t, p, smallMemBytes)
	assertState(t, p, got, stateBusy)
}

// 3. Budget full with two idle cpp20 containers; claiming java17 must evict the
// oldest (LRU) cpp20 before creating the java17 container.
func TestClaim_MemoryFull_EvictsLRUIdleContainer(t *testing.T) {
	var removedID string
	var createdImage string
	mock := &mockDockerClient{
		createFn: func(_ context.Context, opts client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
			createdImage = opts.Config.Image
			return client.ContainerCreateResult{ID: "ctr-new"}, nil
		},
		removeFn: func(_ context.Context, id string, _ client.ContainerRemoveOptions) (client.ContainerRemoveResult, error) {
			removedID = id
			return client.ContainerRemoveResult{}, nil
		},
	}
	p := newTestPool(t, testCfg(2), mock)
	c1 := putIdleContainer(t, p, "cpp20", 2*time.Minute) // older — should be evicted
	putIdleContainer(t, p, "cpp20", 1*time.Minute)       // newer

	got, err := p.Claim(context.Background(), "java17")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected container, got nil")
	}
	if removedID != c1.id {
		t.Errorf("removed %q, want oldest container %q", removedID, c1.id)
	}
	if createdImage != "judge:java17" {
		t.Errorf("created image = %q, want %q", createdImage, "judge:java17")
	}
	if got.language != "java17" {
		t.Errorf("container language = %q, want java17", got.language)
	}
	// evict c1 (-64MB) + create java17 (+64MB) = still 2×64MB
	assertAllocatedBytes(t, p, 2*smallMemBytes)
}

// 4. LRU eviction path: ContainerRemove fails → Claim returns error and
// allocatedBytes is unchanged.
func TestClaim_LRUEviction_DockerFails_ReturnsError(t *testing.T) {
	mock := &mockDockerClient{
		removeFn: func(_ context.Context, _ string, _ client.ContainerRemoveOptions) (client.ContainerRemoveResult, error) {
			return client.ContainerRemoveResult{}, errors.New("docker: daemon unavailable")
		},
	}
	p := newTestPool(t, testCfg(2), mock)
	c1 := putIdleContainer(t, p, "cpp20", 2*time.Minute)
	putIdleContainer(t, p, "cpp20", 1*time.Minute)

	_, err := p.Claim(context.Background(), "java17")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertAllocatedBytes(t, p, 2*smallMemBytes)
	assertState(t, p, c1, stateIdle)
}

// 5. Budget full and the only container is busy: Claim blocks until Release.
func TestClaim_AllBusy_BlocksUntilRelease(t *testing.T) {
	mock := &mockDockerClient{}
	p := newTestPool(t, testCfg(1), mock)
	busy, _ := p.Claim(context.Background(), "cpp20")

	resultCh := make(chan *Container, 1)
	go func() {
		c, _ := p.Claim(context.Background(), "cpp20")
		resultCh <- c
	}()
	time.Sleep(20 * time.Millisecond) // let the goroutine reach the blocked select

	p.Release(busy)

	select {
	case c := <-resultCh:
		if c == nil {
			t.Error("expected container after Release, got nil")
		}
	case <-time.After(time.Second):
		t.Error("Claim did not unblock after Release")
	}
}

// 6. Context cancelled while Claim is blocked → returns ctx.Err().
func TestClaim_ContextCancelled_ReturnsError(t *testing.T) {
	mock := &mockDockerClient{}
	p := newTestPool(t, testCfg(1), mock)
	_, _ = p.Claim(context.Background(), "cpp20") // fill the budget; never released

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, err := p.Claim(ctx, "cpp20")
		errCh <- err
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected Canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Error("Claim did not return after context cancellation")
	}
}

// 7. Release transitions the container to stateIdle and updates lastUsedAt.
func TestRelease_MarksIdleAndUpdatesTimestamp(t *testing.T) {
	mock := &mockDockerClient{}
	p := newTestPool(t, testCfg(1), mock)
	c, _ := p.Claim(context.Background(), "cpp20")

	// Sleep 1ms to ensure the timestamp set by Claim is strictly older than
	// `before`. On Windows, time.Now() has ~15ms resolution so without this
	// sleep both timestamps could be equal, making the check useless.
	time.Sleep(time.Millisecond)
	before := time.Now()
	p.Release(c)

	assertState(t, p, c, stateIdle)
	p.mu.Lock()
	if c.lastUsedAt.Before(before) {
		t.Error("lastUsedAt was not updated on Release")
	}
	p.mu.Unlock()
}

// 8. Reaper destroys an idle container that has exceeded IdleTimeout.
func TestReaper_ExpiredIdleContainer_IsDestroyed(t *testing.T) {
	mock := &mockDockerClient{}
	cfg := PoolConfig{
		BudgetBytes:  10 * smallMemBytes,
		IdleTimeout:  50 * time.Millisecond,
		ReapInterval: 20 * time.Millisecond,
		Languages:    map[string]LanguageConfig{"cpp20": {Image: "judge:cpp20", MemoryBytes: smallMemBytes}},
	}
	p := newTestPool(t, cfg, mock)
	putIdleContainer(t, p, "cpp20", 200*time.Millisecond) // already past IdleTimeout

	time.Sleep(5 * cfg.ReapInterval) // wait for at least one full reap cycle

	if mock.removeCnt.Load() == 0 {
		t.Error("expected ContainerRemove to be called by the reaper")
	}
	assertAllocatedBytes(t, p, 0)
}

// 9. Reaper must not destroy a busy container even if its lastUsedAt is old.
func TestReaper_BusyContainer_NotDestroyed(t *testing.T) {
	mock := &mockDockerClient{}
	cfg := PoolConfig{
		BudgetBytes:  10 * smallMemBytes,
		IdleTimeout:  50 * time.Millisecond,
		ReapInterval: 20 * time.Millisecond,
		Languages:    map[string]LanguageConfig{"cpp20": {Image: "judge:cpp20", MemoryBytes: smallMemBytes}},
	}
	p := newTestPool(t, cfg, mock)
	c := putIdleContainer(t, p, "cpp20", 200*time.Millisecond)
	p.mu.Lock()
	c.state = stateBusy
	p.mu.Unlock()

	time.Sleep(5 * cfg.ReapInterval)

	if mock.removeCnt.Load() != 0 {
		t.Errorf("removeCnt = %d, want 0 — busy container must not be reaped", mock.removeCnt.Load())
	}
	assertAllocatedBytes(t, p, smallMemBytes)
}

// 10. Reaper: ContainerRemove fails → container is restored to stateIdle so the
// next tick can retry.
func TestReaper_DockerFails_ContainerRestoredToIdle(t *testing.T) {
	mock := &mockDockerClient{
		removeFn: func(_ context.Context, _ string, _ client.ContainerRemoveOptions) (client.ContainerRemoveResult, error) {
			return client.ContainerRemoveResult{}, errors.New("docker: daemon unavailable")
		},
	}
	cfg := PoolConfig{
		BudgetBytes:  10 * smallMemBytes,
		IdleTimeout:  50 * time.Millisecond,
		ReapInterval: 20 * time.Millisecond,
		Languages:    map[string]LanguageConfig{"cpp20": {Image: "judge:cpp20", MemoryBytes: smallMemBytes}},
	}
	p := newTestPool(t, cfg, mock)
	c := putIdleContainer(t, p, "cpp20", 200*time.Millisecond)

	time.Sleep(5 * cfg.ReapInterval)

	assertState(t, p, c, stateIdle)
	assertAllocatedBytes(t, p, smallMemBytes)
}

// 11. A container in stateDraining must not be claimable — Claim must skip it
// and block (or time out) instead of returning a container that is mid-removal.
func TestReaper_DrainingContainer_SkippedByClaim(t *testing.T) {
	mock := &mockDockerClient{}
	p := newTestPool(t, testCfg(1), mock)

	// Simulate a container mid-removal (set by the reaper before calling Docker).
	c := &Container{
		id:          "ctr-draining",
		language:    "cpp20",
		memoryBytes: smallMemBytes,
		state:       stateDraining,
		lastUsedAt:  time.Now(),
	}
	p.mu.Lock()
	p.containers = append(p.containers, c)
	p.allocatedBytes += c.memoryBytes
	p.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := p.Claim(ctx, "cpp20")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
	assertState(t, p, c, stateDraining) // must not have been touched by Claim
}

// 12. A draining container does not block Claim if there is still budget available —
// Claim must create a new container instead of waiting for a Release signal.
func TestClaim_DrainingContainerWithBudgetAvailable_CreatesNew(t *testing.T) {
	mock := &mockDockerClient{}
	p := newTestPool(t, testCfg(2), mock) // budget = 2 containers

	// Inject one draining container (consumes 1 slot, leaving 1 free).
	draining := &Container{
		id:          "ctr-draining",
		language:    "cpp20",
		memoryBytes: smallMemBytes,
		state:       stateDraining,
		lastUsedAt:  time.Now(),
	}
	p.mu.Lock()
	p.containers = append(p.containers, draining)
	p.allocatedBytes += draining.memoryBytes
	p.mu.Unlock()

	// Budget still has room — Claim must not block.
	got, err := p.Claim(context.Background(), "cpp20")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected new container, got nil")
	}
	if mock.createCnt.Load() != 1 {
		t.Errorf("createCnt = %d, want 1", mock.createCnt.Load())
	}
	assertAllocatedBytes(t, p, 2*smallMemBytes)
	assertState(t, p, draining, stateDraining) // must not have been touched by Claim
}
