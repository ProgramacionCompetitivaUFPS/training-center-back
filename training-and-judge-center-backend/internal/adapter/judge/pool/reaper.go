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

// evictExpired removes containers that have been idle longer than IdleTimeout, and
// retries removal of containers left in stateDraining by previous failed Discard calls.
//
// Invariant: any container in stateDraining at the start of a tick was left there by a
// previous failed Discard. The reaper either removes it or leaves it to retry on the next
// tick. A concurrent Discard may add new ones during this call.
//
// Two-phase design to keep Docker I/O outside the lock:
//  1. Under lock: collect (a) pre-existing draining containers and (b) expired idle
//     containers — marking the latter stateDraining so Claim skips them during I/O.
//  2. Outside lock: call ContainerRemove for each group.
//     stuck containers: on failure stay in stateDraining (retried next tick).
//     toEvict containers: on failure restore to stateIdle (retried next tick).
//     On success (both groups): update allocatedBytes and remove from slice.
func (p *Pool) evictExpired(ctx context.Context) {
	now := time.Now()

	p.mu.Lock()
	var stuck []*Container
	var toEvict []*Container
	for _, c := range p.containers {
		if c.state == stateDraining {
			stuck = append(stuck, c)
		} else if c.state == stateIdle && now.Sub(c.lastUsedAt) > p.cfg.IdleTimeout {
			c.state = stateDraining
			toEvict = append(toEvict, c)
		}
	}
	p.mu.Unlock()

	for _, c := range stuck {
		_, err := p.docker.ContainerRemove(ctx, c.id, client.ContainerRemoveOptions{Force: true})
		p.mu.Lock()
		if err != nil {
			slog.ErrorContext(ctx, "pool: retry discard failed, reaper will try again",
				"container_id", c.id, "error", err)
		} else {
			p.allocatedBytes -= c.memoryBytes
			p.containers = removeFromSlice(p.containers, c)
		}
		p.mu.Unlock()
	}

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
