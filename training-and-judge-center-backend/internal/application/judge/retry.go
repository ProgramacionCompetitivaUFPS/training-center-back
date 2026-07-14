package judge

import "time"

type RetryConfig struct {
	MaxAttempts int
	BackoffBase time.Duration
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{MaxAttempts: 3, BackoffBase: 500 * time.Millisecond}
}
