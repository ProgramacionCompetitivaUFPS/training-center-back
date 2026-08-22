package pool

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type Pool struct {
	mu             sync.Mutex
	cfg            PoolConfig
	docker         dockerLifecycle
	containers     []*Container
	allocatedBytes int64
	// released is buffered(1) so Release never drops a wakeup signal even when
	// the waiting goroutine has not yet reached the select statement.
	released chan struct{}
	stop     chan struct{}
	done     chan struct{}
	stopOnce sync.Once
	// ctx is cancelled by Stop so that in-flight Docker calls in the reaper
	// are unblocked when the pool is shutting down.
	ctx      context.Context
	cancelFn context.CancelFunc
}

func NewPool(cfg PoolConfig, docker dockerLifecycle) *Pool {
	for lang, lc := range cfg.Languages {
		if lc.MemoryBytes <= 0 {
			panic(fmt.Sprintf("pool: language %q has non-positive MemoryBytes", lang))
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Pool{
		cfg:      cfg,
		docker:   docker,
		released: make(chan struct{}, 1),
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
		ctx:      ctx,
		cancelFn: cancel,
	}
}

func (p *Pool) Start() {
	go p.reap()
}

// Safe to call multiple times.
func (p *Pool) Stop() {
	p.stopOnce.Do(func() {
		p.cancelFn()
		close(p.stop)
	})
	<-p.done
}

// IsHealthy reports whether the Docker daemon backing the pool is reachable.
// A false result means the pool cannot create or run containers anymore.
func (p *Pool) IsHealthy(ctx context.Context) bool {
	_, err := p.docker.Ping(ctx, client.PingOptions{})
	return err == nil
}

// LanguageCeiling asks Claim for the language's own limit, which is what the
// pool charged against its budget. It is what a caller running trusted work —
// compiling, or executing an already-built artifact — wants.
const LanguageCeiling int64 = 0

// The caller must Release the returned container when done.
//
// memoryBytes is the ceiling the kernel will enforce on the returned container;
// zero asks for the language's own. Larger values are capped to it, since that
// is what the pool charged against its budget. Because every claim states its
// ceiling, a container never carries the previous claimer's.
//
// Algorithm:
//  1. Fast path: reuse an idle container of the same language.
//  2. If budget allows, create a new container.
//  3. Otherwise, LRU-evict idle containers until budget allows, then create.
//  4. If no idle containers exist to evict, block until Release signals one.
func (p *Pool) Claim(ctx context.Context, language string, memoryBytes int64) (*Container, error) {
	langCfg, ok := p.cfg.Languages[language]
	if !ok {
		return nil, fmt.Errorf("pool: unknown language %q", language)
	}

	limit := memoryBytes
	if limit > langCfg.MemoryBytes {
		slog.WarnContext(ctx, "pool: memory limit over the language ceiling, capping",
			"language", language, "requested", limit, "ceiling", langCfg.MemoryBytes)
	}
	if limit <= 0 || limit > langCfg.MemoryBytes {
		limit = langCfg.MemoryBytes
	}

retry:
	p.mu.Lock()

	for _, c := range p.containers {
		if c.language == language && c.state == stateIdle {
			c.state = stateBusy
			p.mu.Unlock()
			// Owned now, so limitBytes is ours to read and write without the lock.
			if err := p.applyMemoryLimit(ctx, c, limit); err != nil {
				// A wrong ceiling is a wrong verdict, so the container goes.
				p.Discard(ctx, c)
				return nil, err
			}
			return c, nil
		}
	}

	// Invariant: lock is held at the start of each iteration and at loop exit.
	for !p.canCreate(langCfg) {
		oldest := p.oldestIdleContainer()
		if oldest == nil {
			// All containers are busy — wait for a Release signal.
			p.mu.Unlock()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-p.released:
				goto retry
			}
		}
		// Mark before releasing the lock so Claim skips it during Docker I/O.
		oldest.state = stateDraining
		p.mu.Unlock()

		_, err := p.docker.ContainerRemove(ctx, oldest.id, client.ContainerRemoveOptions{Force: true})

		p.mu.Lock()
		if err != nil {
			oldest.state = stateIdle // restore so the reaper can retry on the next tick
			p.mu.Unlock()
			slog.ErrorContext(ctx, "pool: evict container failed", "container_id", oldest.id, "error", err)
			return nil, fmt.Errorf("pool: evict container %s: %w", oldest.id, err)
		}
		p.allocatedBytes -= oldest.memoryBytes
		p.containers = removeFromSlice(p.containers, oldest)
	}
	p.mu.Unlock()

	// Created straight at the ceiling this claim asked for: no update needed.
	id, err := p.createContainer(ctx, langCfg, limit)
	if err != nil {
		return nil, err
	}

	c := &Container{
		id:          id,
		language:    language,
		memoryBytes: langCfg.MemoryBytes,
		limitBytes:  limit,
		state:       stateBusy,
		lastUsedAt:  time.Now(),
	}

	p.mu.Lock()
	p.containers = append(p.containers, c)
	p.allocatedBytes += langCfg.MemoryBytes
	p.mu.Unlock()

	return c, nil
}

// The channel is buffered(1), so the signal is never lost if no goroutine
// is waiting yet.
func (p *Pool) Release(c *Container) {
	p.mu.Lock()
	c.state = stateIdle
	c.lastUsedAt = time.Now()
	p.mu.Unlock()

	select {
	case p.released <- struct{}{}:
	default:
	}
}

// Caller must hold p.mu.
func (p *Pool) canCreate(langCfg LanguageConfig) bool {
	return p.allocatedBytes+langCfg.MemoryBytes <= p.cfg.BudgetBytes
}

// Caller must hold p.mu.
func (p *Pool) oldestIdleContainer() *Container {
	var oldest *Container
	for _, c := range p.containers {
		if c.state != stateIdle {
			continue
		}
		if oldest == nil || c.lastUsedAt.Before(oldest.lastUsedAt) {
			oldest = c
		}
	}
	return oldest
}

// Removes the dangling container if ContainerStart fails after ContainerCreate succeeds.
// Caller must NOT hold p.mu.
func (p *Pool) createContainer(ctx context.Context, langCfg LanguageConfig, memoryBytes int64) (string, error) {
	result, err := p.docker.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config: &container.Config{
			Image: langCfg.Image,
			// sleep infinity keeps the container alive so the executor can
			// issue docker exec calls without a running service inside.
			Cmd: []string{"sleep", "infinity"},
		},
		HostConfig: &container.HostConfig{
			Resources: container.Resources{
				Memory: memoryBytes,
				// MemorySwap == Memory disables swap entirely, making MLE
				// deterministic: the kernel always sends SIGKILL (exit 137)
				// when the process exceeds the memory limit.
				MemorySwap: memoryBytes,
				NanoCPUs:   parseCPUNanos(langCfg.CPU),
			},
			NetworkMode: "none",
		},
	})
	if err != nil {
		slog.ErrorContext(ctx, "pool: ContainerCreate failed", "image", langCfg.Image, "error", err)
		return "", fmt.Errorf("pool: create container: %w", err)
	}

	_, err = p.docker.ContainerStart(ctx, result.ID, client.ContainerStartOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "pool: ContainerStart failed", "container_id", result.ID, "error", err)
		_, _ = p.docker.ContainerRemove(ctx, result.ID, client.ContainerRemoveOptions{Force: true})
		return "", fmt.Errorf("pool: start container %s: %w", result.ID, err)
	}
	return result.ID, nil
}

// applyMemoryLimit rewrites the cgroup ceiling of a container already running,
// skipping the round trip when it already matches. Only the memory fields go in
// the request: the zero-valued ones leave the CPU cap alone.
// Caller must own the container (stateBusy) and must NOT hold p.mu.
func (p *Pool) applyMemoryLimit(ctx context.Context, c *Container, memoryBytes int64) error {
	if c.limitBytes == memoryBytes {
		return nil
	}
	_, err := p.docker.ContainerUpdate(ctx, c.id, client.ContainerUpdateOptions{
		Resources: &container.Resources{Memory: memoryBytes, MemorySwap: memoryBytes},
	})
	if err != nil {
		slog.ErrorContext(ctx, "pool: ContainerUpdate failed",
			"container_id", c.id, "memory_bytes", memoryBytes, "error", err)
		return fmt.Errorf("pool: set memory limit on %s: %w", c.id, err)
	}
	c.limitBytes = memoryBytes
	return nil
}

// Returns 0 (no CPU limit) if the value is empty or cannot be parsed.
func parseCPUNanos(cpu string) int64 {
	if cpu == "" {
		return 0
	}
	f, err := strconv.ParseFloat(cpu, 64)
	if err != nil || f <= 0 {
		return 0
	}
	return int64(f * 1e9)
}

// Discard destroys a busy container that is in an unknown state (e.g. safety-net fired).
// The caller must NOT call Release after Discard.
//
// If ContainerRemove fails, the container stays in stateDraining with allocatedBytes
// unchanged — the real memory is still occupied. evictExpired retries on the next tick.
func (p *Pool) Discard(ctx context.Context, c *Container) {
	p.mu.Lock()
	c.state = stateDraining
	p.mu.Unlock()

	_, err := p.docker.ContainerRemove(ctx, c.id, client.ContainerRemoveOptions{Force: true})
	if err != nil {
		slog.ErrorContext(ctx, "pool: discard container failed, reaper will retry",
			"container_id", c.id, "error", err)
		return
	}

	p.mu.Lock()
	p.allocatedBytes -= c.memoryBytes
	p.containers = removeFromSlice(p.containers, c)
	p.mu.Unlock()
}

func removeFromSlice(s []*Container, target *Container) []*Container {
	for i, c := range s {
		if c == target {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}
