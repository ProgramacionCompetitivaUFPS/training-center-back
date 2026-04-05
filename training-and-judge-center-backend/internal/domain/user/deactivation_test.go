package user

import (
	"testing"
	"time"
)

func TestRegisterFailure(t *testing.T) {
	newRequest := func(attempts int) *DeactivationRequest {
		return RestoreDeactivationRequest(
			"req-id", "user-id", "123456",
			time.Now().Add(time.Hour),
			attempts, nil,
			DeactivationStatusPending,
			time.Now(), time.Now(),
		)
	}

	tests := []struct {
		name            string
		initialAttempts int
		expectedStatus  DeactivationStatus
		expectBlocked   bool
	}{
		{"first failure stays pending", 0, DeactivationStatusPending, false},
		{"second failure stays pending", 1, DeactivationStatusPending, false},
		{"third failure stays pending", 2, DeactivationStatusPending, false},
		{"fourth failure stays pending", 3, DeactivationStatusPending, false},
		{"exactly at threshold becomes blocked", 4, DeactivationStatusBlocked, true},
		{"above threshold stays blocked", 5, DeactivationStatusBlocked, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now()
			req := newRequest(tt.initialAttempts)

			req.RegisterFailure()

			if req.attempts != tt.initialAttempts+1 {
				t.Errorf("attempts: got %d, want %d", req.attempts, tt.initialAttempts+1)
			}
			if req.status != tt.expectedStatus {
				t.Errorf("status: got %q, want %q", req.status, tt.expectedStatus)
			}
			if tt.expectBlocked {
				if req.blockedUntil == nil {
					t.Fatal("blockedUntil: got nil, want non-nil")
				}
				expectedBlockedUntil := before.Add(DeactivationBlockDuration)
				if req.blockedUntil.Before(expectedBlockedUntil) {
					t.Errorf("blockedUntil %v is earlier than expected %v", req.blockedUntil, expectedBlockedUntil)
				}
			} else {
				if req.blockedUntil != nil {
					t.Errorf("blockedUntil: got %v, want nil", req.blockedUntil)
				}
			}
			if req.updatedAt.Before(before) {
				t.Errorf("updatedAt %v was not refreshed", req.updatedAt)
			}
		})
	}
}
