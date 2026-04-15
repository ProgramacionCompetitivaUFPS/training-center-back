package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type SessionInvalidator struct {
	client   *redis.Client
	tokenTTL time.Duration
}

func NewSessionInvalidator(client *redis.Client, tokenTTL time.Duration) *SessionInvalidator {
	return &SessionInvalidator{client: client, tokenTTL: tokenTTL}
}

func (s *SessionInvalidator) InvalidateAllUserSessions(ctx context.Context, userID string, timestamp time.Time) error {
	key := fmt.Sprintf("revoked_sessions:%s", userID)
	// TTL matches the JWT expiration so the revocation key outlives any token it covers.
	err := s.client.Set(ctx, key, timestamp.Unix(), s.tokenTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to save session revocation timestamp: %w", err)
	}
	return nil
}

func (s *SessionInvalidator) IsSessionRevoked(ctx context.Context, userID string, tokenIssuedAt time.Time) (bool, error) {
	key := fmt.Sprintf("revoked_sessions:%s", userID)

	val, err := s.client.Get(ctx, key).Int64()
	if err != nil {
		if err == redis.Nil {
			return false, nil // Not revoked
		}
		return false, fmt.Errorf("failed to check session revocation: %w", err)
	}

	// Token issued before or at the time of revocation
	if tokenIssuedAt.Unix() <= val {
		return true, nil
	}

	return false, nil
}
