package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type SessionInvalidator struct {
	client *redis.Client
}

func NewSessionInvalidator(client *redis.Client) *SessionInvalidator {
	return &SessionInvalidator{client: client}
}

func (s *SessionInvalidator) InvalidateAllUserSessions(ctx context.Context, userID string, timestamp time.Time) error {
	key := fmt.Sprintf("revoked_sessions:%s", userID)
	// Guaranteed 24-h expiration (or matching max token lifecycle)
	err := s.client.Set(ctx, key, timestamp.Unix(), 24*time.Hour).Err()
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
