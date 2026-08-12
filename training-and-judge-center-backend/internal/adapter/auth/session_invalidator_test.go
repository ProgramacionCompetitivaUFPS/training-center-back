package auth

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestSessionInvalidator(t *testing.T) (*RedisSessionInvalidator, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewRedisSessionInvalidator(client, time.Hour), mr
}

// The invalidation mark is terminal: once a family is revoked, no legitimate token can
// ever be minted with that sid again, so IsSessionInvalidated ignores iat entirely.
func TestIsSessionInvalidated_MarkExists_ReturnsTrue(t *testing.T) {
	s, _ := newTestSessionInvalidator(t)
	ctx := context.Background()

	if err := s.InvalidateSession(ctx, "family-1", time.Now()); err != nil {
		t.Fatalf("InvalidateSession: unexpected error: %v", err)
	}

	invalidated, err := s.IsSessionInvalidated(ctx, "family-1")
	if err != nil {
		t.Fatalf("IsSessionInvalidated: unexpected error: %v", err)
	}
	if !invalidated {
		t.Error("expected the session to be invalidated once the mark exists")
	}
}

func TestIsSessionInvalidated_NoMark_ReturnsFalse(t *testing.T) {
	s, _ := newTestSessionInvalidator(t)

	invalidated, err := s.IsSessionInvalidated(context.Background(), "family-unknown")
	if err != nil {
		t.Fatalf("IsSessionInvalidated: unexpected error: %v", err)
	}
	if invalidated {
		t.Error("expected false when no invalidation mark exists")
	}
}

func TestInvalidateSession_MarkExpiresWithTokenTTL(t *testing.T) {
	s, mr := newTestSessionInvalidator(t)
	ctx := context.Background()

	if err := s.InvalidateSession(ctx, "family-1", time.Now()); err != nil {
		t.Fatalf("InvalidateSession: unexpected error: %v", err)
	}

	if ttl := mr.TTL("revoked_session_family:family-1"); ttl != time.Hour {
		t.Errorf("mark TTL: got %v, want %v", ttl, time.Hour)
	}
}
