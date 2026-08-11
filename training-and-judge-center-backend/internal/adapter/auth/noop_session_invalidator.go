package auth

import (
	"context"
	"time"
)

type NoOpSessionInvalidator struct{}

func (n *NoOpSessionInvalidator) InvalidateAllUserSessions(_ context.Context, _ string, _ time.Time) error {
	return nil
}

func (n *NoOpSessionInvalidator) IsAllUserSessionRevoked(_ context.Context, _ string, _ time.Time) (bool, error) {
	return false, nil
}

func (n *NoOpSessionInvalidator) InvalidateSession(_ context.Context, _ string, _ time.Time) error {
	return nil
}

func (n *NoOpSessionInvalidator) IsSessionInvalidated(_ context.Context, _ string, _ time.Time) (bool, error) {
	return false, nil
}
