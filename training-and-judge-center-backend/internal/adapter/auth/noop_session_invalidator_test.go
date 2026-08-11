package auth

import (
	"context"
	"testing"
	"time"
)

func TestNoOpSessionInvalidator_AlwaysNeutral(t *testing.T) {
	n := &NoOpSessionInvalidator{}
	ctx := context.Background()
	now := time.Now()

	if err := n.InvalidateAllUserSessions(ctx, "user-1", now); err != nil {
		t.Errorf("InvalidateAllUserSessions: expected nil error, got %v", err)
	}

	revoked, err := n.IsAllUserSessionRevoked(ctx, "user-1", now)
	if err != nil {
		t.Errorf("IsAllUserSessionRevoked: expected nil error, got %v", err)
	}
	if revoked {
		t.Error("IsAllUserSessionRevoked: expected false, got true")
	}

	if err := n.InvalidateSession(ctx, "session-1", now); err != nil {
		t.Errorf("InvalidateSession: expected nil error, got %v", err)
	}

	invalidated, err := n.IsSessionInvalidated(ctx, "session-1", now)
	if err != nil {
		t.Errorf("IsSessionInvalidated: expected nil error, got %v", err)
	}
	if invalidated {
		t.Error("IsSessionInvalidated: expected false, got true")
	}
}
