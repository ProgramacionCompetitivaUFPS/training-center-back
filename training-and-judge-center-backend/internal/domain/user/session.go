package user

import (
	"context"
	"time"
)

type SessionInvalidator interface {
	InvalidateAllUserSessions(ctx context.Context, userID string, timestamp time.Time) error
	IsSessionRevoked(ctx context.Context, userID string, tokenIssuedAt time.Time) (bool, error)
}
