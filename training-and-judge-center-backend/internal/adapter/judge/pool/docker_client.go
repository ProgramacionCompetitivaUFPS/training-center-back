package pool

import (
	"context"

	"github.com/moby/moby/client"
)

// *client.Client from github.com/moby/moby/client satisfies this interface.
type dockerLifecycle interface {
	ContainerCreate(ctx context.Context, opts client.ContainerCreateOptions) (client.ContainerCreateResult, error)
	ContainerStart(ctx context.Context, containerID string, opts client.ContainerStartOptions) (client.ContainerStartResult, error)
	ContainerRemove(ctx context.Context, containerID string, opts client.ContainerRemoveOptions) (client.ContainerRemoveResult, error)
	ContainerUpdate(ctx context.Context, containerID string, opts client.ContainerUpdateOptions) (client.ContainerUpdateResult, error)
	Ping(ctx context.Context, options client.PingOptions) (client.PingResult, error)
}
