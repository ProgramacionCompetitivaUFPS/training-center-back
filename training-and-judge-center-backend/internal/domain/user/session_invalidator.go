package user

import (
	"context"
	"time"
)

type SessionInvalidator interface {
	InvalidateAllUserSessions(ctx context.Context, userID string, timestamp time.Time) error
	IsAllUserSessionRevoked(ctx context.Context, userID string, tokenIssuedAt time.Time) (bool, error)

	InvalidateSession(ctx context.Context, sessionID string, timestamp time.Time) error
	IsSessionInvalidated(ctx context.Context, sessionID string, tokenIssuedAt time.Time) (bool, error)
}
