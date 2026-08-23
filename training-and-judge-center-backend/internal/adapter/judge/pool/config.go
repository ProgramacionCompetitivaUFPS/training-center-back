package pool

import "time"

// SharedVolumePath is where the judging volume is mounted, both inside every
// container and in the worker itself. Only the daemon's source for that mount
// differs between topologies, so the target stays a constant.
const SharedVolumePath = "/judging"

type LanguageConfig struct {
	Image       string
	CPU         string
	MemoryBytes int64
}

type PoolConfig struct {
	// BudgetBytes is this pool's share of container memory. Pools draw from the
	// same Docker daemon, so the split between them is decided by the caller.
	BudgetBytes  int64
	IdleTimeout  time.Duration
	ReapInterval time.Duration
	Languages    map[string]LanguageConfig
	// SharedVolumeSource is what the daemon resolves the mount against: a path
	// inside the dind sidecar, or a Docker volume name when the daemon is the
	// host's. A mount cannot change after creation, so it belongs to the pool.
	SharedVolumeSource string
	// SharedVolumeReadOnly keeps a pool that only reads from writing: a checker
	// that guessed another judging's directory could otherwise rewrite it.
	SharedVolumeReadOnly bool
}
