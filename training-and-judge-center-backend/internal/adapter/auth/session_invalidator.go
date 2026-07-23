package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RedisSessionInvalidator struct {
	client   *redis.Client
	tokenTTL time.Duration
}

func NewRedisSessionInvalidator(client *redis.Client, tokenTTL time.Duration) *RedisSessionInvalidator {
	return &RedisSessionInvalidator{client: client, tokenTTL: tokenTTL}
}

func (s *RedisSessionInvalidator) InvalidateAllUserSessions(ctx context.Context, userID string, timestamp time.Time) error {
	key := fmt.Sprintf("revoked_sessions:%s", userID)
	// TTL matches the JWT expiration so the revocation key outlives any token it covers.
	if err := s.client.Set(ctx, key, timestamp.Unix(), s.tokenTTL).Err(); err != nil {
		slog.ErrorContext(ctx, "failed to save session revocation timestamp", "user_id", userID, "error", err)
		return apperror.NewInternal()
	}
	return nil
}

func (s *RedisSessionInvalidator) IsSessionRevoked(ctx context.Context, userID string, tokenIssuedAt time.Time) (bool, error) {
	key := fmt.Sprintf("revoked_sessions:%s", userID)

	val, err := s.client.Get(ctx, key).Int64()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		slog.ErrorContext(ctx, "failed to check session revocation", "user_id", userID, "error", err)
		return false, apperror.NewInternal()
	}

	return tokenIssuedAt.Unix() <= val, nil
}
