package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/training-judge-center/backend/pkg/apperror"
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

var revokedAt = time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)

func assertInternalError(t *testing.T, err error) {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindInternal {
		t.Errorf("expected an internal apperror, got %v", err)
	}
}

func TestIsAllUserSessionRevoked(t *testing.T) {
	tests := []struct {
		name     string
		issuedAt time.Time
		want     bool
	}{
		{"a token issued before the revocation is revoked", revokedAt.Add(-time.Second), true},
		{"a token issued in the same second as the revocation is revoked", revokedAt, true},
		{"a token issued after the revocation is still valid", revokedAt.Add(time.Second), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := newTestSessionInvalidator(t)
			ctx := context.Background()
			if err := s.InvalidateAllUserSessions(ctx, "user-1", revokedAt); err != nil {
				t.Fatalf("InvalidateAllUserSessions: %v", err)
			}

			got, err := s.IsAllUserSessionRevoked(ctx, "user-1", tt.issuedAt)

			if err != nil {
				t.Fatalf("IsAllUserSessionRevoked: %v", err)
			}
			if got != tt.want {
				t.Errorf("revoked = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAllUserSessionRevoked_NoMark_ReturnsFalse(t *testing.T) {
	s, _ := newTestSessionInvalidator(t)

	got, err := s.IsAllUserSessionRevoked(context.Background(), "user-unknown", revokedAt)

	if err != nil {
		t.Fatalf("IsAllUserSessionRevoked: %v", err)
	}
	if got {
		t.Error("expected false when the user has no revocation mark")
	}
}

func TestIsAllUserSessionRevoked_AnotherUsersRevocationDoesNotApply(t *testing.T) {
	s, _ := newTestSessionInvalidator(t)
	ctx := context.Background()
	if err := s.InvalidateAllUserSessions(ctx, "user-1", revokedAt); err != nil {
		t.Fatalf("InvalidateAllUserSessions: %v", err)
	}

	got, err := s.IsAllUserSessionRevoked(ctx, "user-2", revokedAt.Add(-time.Hour))

	if err != nil {
		t.Fatalf("IsAllUserSessionRevoked: %v", err)
	}
	if got {
		t.Error("revoking user-1 must not revoke user-2")
	}
}

func TestInvalidateAllUserSessions_StoresTheTimestampWithTheTokenTTL(t *testing.T) {
	s, mr := newTestSessionInvalidator(t)

	if err := s.InvalidateAllUserSessions(context.Background(), "user-1", revokedAt); err != nil {
		t.Fatalf("InvalidateAllUserSessions: %v", err)
	}

	if got, _ := mr.Get("revoked_sessions:user-1"); got != "1736942400" {
		t.Errorf("stored value: got %q, want %q", got, "1736942400")
	}
	if ttl := mr.TTL("revoked_sessions:user-1"); ttl != time.Hour {
		t.Errorf("mark TTL: got %v, want %v", ttl, time.Hour)
	}
}

func TestIsAllUserSessionRevoked_CorruptMark_ReturnsInternal(t *testing.T) {
	s, mr := newTestSessionInvalidator(t)
	if err := mr.Set("revoked_sessions:user-1", "not-a-timestamp"); err != nil {
		t.Fatal(err)
	}

	got, err := s.IsAllUserSessionRevoked(context.Background(), "user-1", revokedAt)

	assertInternalError(t, err)
	if got {
		t.Error("a corrupt mark must not be read as a revocation")
	}
}

func TestSessionInvalidator_RedisDown_ReturnsInternal(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		call func(s *RedisSessionInvalidator) error
	}{
		{"invalidating all of a user's sessions", func(s *RedisSessionInvalidator) error {
			return s.InvalidateAllUserSessions(ctx, "user-1", revokedAt)
		}},
		{"checking a user's revocation", func(s *RedisSessionInvalidator) error {
			_, err := s.IsAllUserSessionRevoked(ctx, "user-1", revokedAt)
			return err
		}},
		{"invalidating one session", func(s *RedisSessionInvalidator) error {
			return s.InvalidateSession(ctx, "family-1", revokedAt)
		}},
		{"checking one session", func(s *RedisSessionInvalidator) error {
			_, err := s.IsSessionInvalidated(ctx, "family-1")
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := miniredis.RunT(t)
			client := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1})
			t.Cleanup(func() { _ = client.Close() })
			s := NewRedisSessionInvalidator(client, time.Hour)
			mr.Close()

			assertInternalError(t, tt.call(s))
		})
	}
}
