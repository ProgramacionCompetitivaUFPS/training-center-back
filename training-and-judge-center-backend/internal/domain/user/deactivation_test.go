package user_test

import (
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/user"
)

func TestRestoreDeactivationRequest_BlockedWithNilBlockedUntil(t *testing.T) {
	req := user.RestoreDeactivationRequest(
		"req-id", "user-id", "123456",
		testNow.Add(time.Hour),
		5, nil,
		user.DeactivationStatusBlocked,
		testNow, testNow,
	)

	if req.Status() != user.DeactivationStatusExpired {
		t.Errorf("expected status EXPIRED when restored as BLOCKED with nil blockedUntil, got %q", req.Status())
	}
}

func TestIsCurrentlyBlocked(t *testing.T) {
	tests := []struct {
		name         string
		status       user.DeactivationStatus
		blockedUntil *time.Time
		expected     bool
	}{
		{
			name:         "blocked and within period",
			status:       user.DeactivationStatusBlocked,
			blockedUntil: ptr(testNow.Add(30 * time.Minute)),
			expected:     true,
		},
		{
			name:         "blocked but period expired",
			status:       user.DeactivationStatusBlocked,
			blockedUntil: ptr(testNow.Add(-1 * time.Minute)),
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
				testNow.Add(time.Hour),
				0, tt.blockedUntil,
				tt.status,
				testNow, testNow,
			)
			got := req.IsCurrentlyBlocked(testNow)
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
			testNow.Add(time.Hour),
			attempts, nil,
			user.DeactivationStatusPending,
			testNow, testNow,
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
			req := newRequest(tt.initialAttempts)

			req.RegisterFailure(testNow)

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
				expectedBlockedUntil := testNow.Add(user.DeactivationBlockDuration)
				if req.BlockedUntil().Before(expectedBlockedUntil) {
					t.Errorf("blockedUntil %v is earlier than expected %v", req.BlockedUntil(), expectedBlockedUntil)
				}
			} else {
				if req.BlockedUntil() != nil {
					t.Errorf("blockedUntil: got %v, want nil", req.BlockedUntil())
				}
			}
			if req.UpdatedAt().Before(testNow) {
				t.Errorf("updatedAt %v was not refreshed", req.UpdatedAt())
			}
		})
	}
}

func TestNewDeactivationRequest_StartsPendingAndExpiresInFifteenMinutes(t *testing.T) {
	req, err := user.NewDeactivationRequest("req-1", "user-1", "123456", localNow)

	if err != nil {
		t.Fatalf("NewDeactivationRequest: %v", err)
	}
	if req.ID() != "req-1" || req.UserID() != "user-1" || req.VerificationCode() != "123456" {
		t.Errorf("fields = %s/%s/%s, want req-1/user-1/123456", req.ID(), req.UserID(), req.VerificationCode())
	}
	if req.Status() != user.DeactivationStatusPending || req.Attempts() != 0 || req.BlockedUntil() != nil {
		t.Errorf("state = %s/%d/%v, want PENDING with no attempts and no block", req.Status(), req.Attempts(), req.BlockedUntil())
	}
	assertInUTC(t, "CreatedAt", req.CreatedAt(), testNow)
	assertInUTC(t, "UpdatedAt", req.UpdatedAt(), testNow)
	assertInUTC(t, "ExpiresAt", req.ExpiresAt(), time.Date(2024, 1, 15, 12, 15, 0, 0, time.UTC))
}

func TestNewDeactivationRequest_EmptyID_IsAnInternalError(t *testing.T) {
	req, err := user.NewDeactivationRequest("", "user-1", "123456", testNow)

	assertInternalError(t, "empty id", err)
	if req != nil {
		t.Errorf("expected no request, got %+v", req)
	}
}

func TestDeactivationRequest_IsExpired(t *testing.T) {
	expiresAt := testNow.Add(15 * time.Minute)
	tests := []struct {
		name   string
		now    time.Time
		status user.DeactivationStatus
		want   bool
	}{
		{"well before the expiry", testNow, user.DeactivationStatusPending, false},
		{"well after the expiry", expiresAt.Add(time.Minute), user.DeactivationStatusPending, true},
		{"marked as expired even before the time", testNow, user.DeactivationStatusExpired, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := user.RestoreDeactivationRequest("req-1", "user-1", "123456", expiresAt, 0, nil, tt.status, testNow, testNow)

			if got := req.IsExpired(tt.now); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeactivationRequest_MarkAsExpiredAndConfirm_SetTheStatusAndTheInstantInUTC(t *testing.T) {
	tests := []struct {
		name string
		call func(r *user.DeactivationRequest)
		want user.DeactivationStatus
	}{
		{"marking it as expired", func(r *user.DeactivationRequest) { r.MarkAsExpired(localNow) }, user.DeactivationStatusExpired},
		{"confirming it", func(r *user.DeactivationRequest) { r.Confirm(localNow) }, user.DeactivationStatusConfirmed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := user.NewDeactivationRequest("req-1", "user-1", "123456", testNow.Add(-time.Hour))
			if err != nil {
				t.Fatal(err)
			}

			tt.call(req)

			if req.Status() != tt.want {
				t.Errorf("Status() = %s, want %s", req.Status(), tt.want)
			}
			assertInUTC(t, "UpdatedAt", req.UpdatedAt(), testNow)
		})
	}
}

func TestDeactivationRequest_IsBlocked_FollowsTheStatusOnly(t *testing.T) {
	blockedUntil := testNow.Add(-time.Hour)
	blocked := user.RestoreDeactivationRequest("req-1", "user-1", "123456", testNow, 5, &blockedUntil, user.DeactivationStatusBlocked, testNow, testNow)
	pending := user.RestoreDeactivationRequest("req-2", "user-1", "123456", testNow, 0, nil, user.DeactivationStatusPending, testNow, testNow)

	if !blocked.IsBlocked() {
		t.Error("IsBlocked() = false for a BLOCKED request, want true even once the block period has passed")
	}
	if pending.IsBlocked() {
		t.Error("IsBlocked() = true for a PENDING request, want false")
	}
}

func TestDeactivationRequest_BlockedUntil_ReturnsACopy(t *testing.T) {
	blockedUntil := testNow.Add(time.Hour)
	req := user.RestoreDeactivationRequest("req-1", "user-1", "123456", testNow, 5, &blockedUntil, user.DeactivationStatusBlocked, testNow, testNow)

	*req.BlockedUntil() = testNow

	if got := req.BlockedUntil(); got == nil || !got.Equal(testNow.Add(time.Hour)) {
		t.Errorf("BlockedUntil() = %v, want %v: writing through the returned pointer must not change the request", got, testNow.Add(time.Hour))
	}
}
