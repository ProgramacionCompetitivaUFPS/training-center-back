package pool

import "time"

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
}
