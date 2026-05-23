package pool

import "time"

type LanguageConfig struct {
	Image       string
	CPU         string
	MemoryBytes int64
}

type PoolConfig struct {
	MemLimitBytes int64         // set from POD_MEMORY_LIMIT (K8s Downward API)
	OverheadBytes int64         // set from POD_MEMORY_OVERHEAD
	IdleTimeout   time.Duration
	ReapInterval  time.Duration
	Languages     map[string]LanguageConfig
}
