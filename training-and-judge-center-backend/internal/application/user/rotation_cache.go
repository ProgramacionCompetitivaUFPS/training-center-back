package user

import (
	"context"
	"time"
)

type RotationCache interface {
	Save(ctx context.Context, oldTokenHash string, output RefreshOutput, ttl time.Duration) error
	Get(ctx context.Context, oldTokenHash string) (*RefreshOutput, error)
}
