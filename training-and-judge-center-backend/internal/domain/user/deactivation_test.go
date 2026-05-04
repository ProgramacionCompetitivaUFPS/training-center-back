package user_test

import (
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/user"
)

func TestRestoreDeactivationRequest_BlockedWithNilBlockedUntil(t *testing.T) {
	req := user.RestoreDeactivationRequest(
		"req-id", "user-id", "123456",
		time.Now().Add(time.Hour),
		5, nil,
		user.DeactivationStatusBlocked,
		time.Now(), time.Now(),
	)

	if req.Status() != user.DeactivationStatusExpired {
		t.Errorf("expected status EXPIRED when restored as BLOCKED with nil blockedUntil, got %q", req.Status())
	}
}

func TestIsCurrentlyBlocked(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name         string
		status       user.DeactivationStatus
		blockedUntil *time.Time
		expected     bool
	}{
		{
			name:         "blocked and within period",
			status:       user.DeactivationStatusBlocked,
			blockedUntil: ptr(now.Add(30 * time.Minute)),
			expected:     true,
		},
		{
			name:         "blocked but period expired",
			status:       user.DeactivationStatusBlocked,
			blockedUntil: ptr(now.Add(-1 * time.Minute)),
			expected:     false,
		},
		{
			name:         "not blocked",
			status:       user.DeactivationStatusPending,
			blockedUntil: nil,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := user.RestoreDeactivationRequest(
				"req-id", "user-id", "123456",
				now.Add(time.Hour),
				0, tt.blockedUntil,
				tt.status,
				now, now,
			)
			got := req.IsCurrentlyBlocked(now)
			if got != tt.expected {
				t.Errorf("IsCurrentlyBlocked: got %v, want %v", got, tt.expected)
			}
		})
	}
}

func ptr(t time.Time) *time.Time { return &t }

func TestRegisterFailure(t *testing.T) {
	newRequest := func(attempts int) *user.DeactivationRequest {
		return user.RestoreDeactivationRequest(
			"req-id", "user-id", "123456",
			time.Now().Add(time.Hour),
			attempts, nil,
			user.DeactivationStatusPending,
			time.Now(), time.Now(),
		)
	}

	tests := []struct {
		name            string
		initialAttempts int
		expectedStatus  user.DeactivationStatus
		expectBlocked   bool
	}{
		{"first failure stays pending", 0, user.DeactivationStatusPending, false},
		{"second failure stays pending", 1, user.DeactivationStatusPending, false},
		{"third failure stays pending", 2, user.DeactivationStatusPending, false},
		{"fourth failure stays pending", 3, user.DeactivationStatusPending, false},
		{"exactly at threshold becomes blocked", 4, user.DeactivationStatusBlocked, true},
		{"above threshold stays blocked", 5, user.DeactivationStatusBlocked, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now()
			req := newRequest(tt.initialAttempts)

			req.RegisterFailure(before)

			if req.Attempts() != tt.initialAttempts+1 {
				t.Errorf("attempts: got %d, want %d", req.Attempts(), tt.initialAttempts+1)
			}
			if req.Status() != tt.expectedStatus {
				t.Errorf("status: got %q, want %q", req.Status(), tt.expectedStatus)
			}
			if tt.expectBlocked {
				if req.BlockedUntil() == nil {
					t.Fatal("blockedUntil: got nil, want non-nil")
				}
				expectedBlockedUntil := before.Add(user.DeactivationBlockDuration)
				if req.BlockedUntil().Before(expectedBlockedUntil) {
					t.Errorf("blockedUntil %v is earlier than expected %v", req.BlockedUntil(), expectedBlockedUntil)
				}
			} else {
				if req.BlockedUntil() != nil {
					t.Errorf("blockedUntil: got %v, want nil", req.BlockedUntil())
				}
			}
			if req.UpdatedAt().Before(before) {
				t.Errorf("updatedAt %v was not refreshed", req.UpdatedAt())
			}
		})
	}
}
