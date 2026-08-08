package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var _ appuser.RotationCache = (*RedisRotationCache)(nil)

type RedisRotationCache struct {
	client *redis.Client
}

func NewRedisRotationCache(client *redis.Client) *RedisRotationCache {
	return &RedisRotationCache{client: client}
}

func (c *RedisRotationCache) Save(ctx context.Context, oldTokenHash string, output appuser.RefreshOutput, ttl time.Duration) error {
	payload, err := json.Marshal(output)
	if err != nil {
		slog.ErrorContext(ctx, "failed to marshal cached refresh output", "error", err)
		return apperror.NewInternal()
	}
	if err := c.client.Set(ctx, "rotation:"+oldTokenHash, payload, ttl).Err(); err != nil {
		slog.ErrorContext(ctx, "failed to cache refresh rotation", "error", err)
		return apperror.NewInternal()
	}
	return nil
}

func (c *RedisRotationCache) Get(ctx context.Context, oldTokenHash string) (*appuser.RefreshOutput, error) {
	val, err := c.client.Get(ctx, "rotation:"+oldTokenHash).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "failed to read cached refresh rotation", "error", err)
		return nil, apperror.NewInternal()
	}
	var output appuser.RefreshOutput
	if err := json.Unmarshal(val, &output); err != nil {
		slog.ErrorContext(ctx, "failed to unmarshal cached refresh output", "error", err)
		return nil, apperror.NewInternal()
	}
	return &output, nil
}
