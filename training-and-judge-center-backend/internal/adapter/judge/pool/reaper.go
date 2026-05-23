package pool

import (
	"context"
	"log/slog"
	"time"

	"github.com/moby/moby/client"
)

func (p *Pool) reap() {
	defer close(p.done)
	ticker := time.NewTicker(p.cfg.ReapInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stop:
			return
		case <-ticker.C:
			p.evictExpired(p.ctx)
		}
	}
}

// evictExpired removes containers that have been idle longer than IdleTimeout.
//
// Two-phase design to keep Docker I/O outside the lock:
//  1. Under lock: collect expired idle containers and mark them stateDraining
//     so Claim skips them during the Docker call window.
//  2. Outside lock: call ContainerRemove for each.
//     On success: update allocatedBytes and remove from slice.
//     On failure: restore to stateIdle so the next tick retries.
func (p *Pool) evictExpired(ctx context.Context) {
	now := time.Now()

	p.mu.Lock()
	var toEvict []*Container
	for _, c := range p.containers {
		if c.state == stateIdle && now.Sub(c.lastUsedAt) > p.cfg.IdleTimeout {
			c.state = stateDraining
			toEvict = append(toEvict, c)
		}
	}
	p.mu.Unlock()

	for _, c := range toEvict {
		_, err := p.docker.ContainerRemove(ctx, c.id, client.ContainerRemoveOptions{Force: true})

		p.mu.Lock()
		if err != nil {
			slog.ErrorContext(ctx, "pool: reaper failed to remove container, will retry next tick",
				"container_id", c.id, "error", err)
			c.state = stateIdle
		} else {
			p.allocatedBytes -= c.memoryBytes
			p.containers = removeFromSlice(p.containers, c)
		}
		p.mu.Unlock()
	}
}
