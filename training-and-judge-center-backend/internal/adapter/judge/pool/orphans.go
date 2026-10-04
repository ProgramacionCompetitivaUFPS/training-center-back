package pool

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/moby/moby/client"
)

// Every container the pool creates carries this label. The worker's daemon is its
// own under Kubernetes (the dind sidecar) but the host's in the local compose, so
// what tells a leftover sandbox from anything else is the label, never the image.
const (
	SandboxLabelKey   = "com.trainingcenter.role"
	SandboxLabelValue = "judge-sandbox"
)

// *client.Client satisfies it.
type dockerCleaner interface {
	ContainerList(ctx context.Context, options client.ContainerListOptions) (client.ContainerListResult, error)
	ContainerRemove(ctx context.Context, containerID string, options client.ContainerRemoveOptions) (client.ContainerRemoveResult, error)
}

// RemoveOrphans force-removes every labelled sandbox container, running or not.
// Call it once at startup, before any pool exists: whatever carries the label then
// belongs to a worker process that is gone. It returns how many it removed.
// This holds only while the daemon is exclusive to one worker; two workers on the same daemon would delete each other's live sandboxes.
func RemoveOrphans(ctx context.Context, docker dockerCleaner) (int, error) {
	list, err := docker.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: make(client.Filters).Add("label", SandboxLabelKey+"="+SandboxLabelValue),
	})
	if err != nil {
		return 0, fmt.Errorf("pool: list leftover sandboxes: %w", err)
	}

	removed := 0
	for _, c := range list.Items {
		if _, err := docker.ContainerRemove(ctx, c.ID, client.ContainerRemoveOptions{Force: true}); err != nil {
			slog.ErrorContext(ctx, "pool: failed to remove a leftover sandbox", "container_id", c.ID, "error", err)
			continue
		}
		removed++
	}
	return removed, nil
}
