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

func TestIsSessionInvalidated_TokenIssuedAfterMark_StillInvalidated(t *testing.T) {
	s, _ := newTestSessionInvalidator(t)
	ctx := context.Background()
	mark := time.Now()

	if err := s.InvalidateSession(ctx, "family-1", mark); err != nil {
		t.Fatalf("InvalidateSession: unexpected error: %v", err)
	}

	invalidated, err := s.IsSessionInvalidated(ctx, "family-1", mark.Add(5*time.Second))
	if err != nil {
		t.Fatalf("IsSessionInvalidated: unexpected error: %v", err)
	}
	if !invalidated {
		t.Error("expected a token issued after the invalidation mark to be rejected")
	}
}

func TestIsSessionInvalidated_TokenIssuedBeforeMark_Invalidated(t *testing.T) {
	s, _ := newTestSessionInvalidator(t)
	ctx := context.Background()
	mark := time.Now()

	if err := s.InvalidateSession(ctx, "family-1", mark); err != nil {
		t.Fatalf("InvalidateSession: unexpected error: %v", err)
	}

	invalidated, err := s.IsSessionInvalidated(ctx, "family-1", mark.Add(-5*time.Second))
	if err != nil {
		t.Fatalf("IsSessionInvalidated: unexpected error: %v", err)
	}
	if !invalidated {
		t.Error("expected a token issued before the invalidation mark to be rejected")
	}
}

func TestIsSessionInvalidated_NoMark_ReturnsFalse(t *testing.T) {
	s, _ := newTestSessionInvalidator(t)

	invalidated, err := s.IsSessionInvalidated(context.Background(), "family-unknown", time.Now())
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
