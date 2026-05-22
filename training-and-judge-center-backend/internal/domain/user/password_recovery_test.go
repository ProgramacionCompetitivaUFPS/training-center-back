package user_test

import (
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/user"
)

func TestPasswordRecoveryRequest_MarkAsUsed(t *testing.T) {
	newReq := func(status user.RequestStatus) *user.PasswordRecoveryRequest {
		return user.RestorePasswordRecoveryRequest(
			"req-id", "user-id", "code123",
			status, testNow.Add(time.Hour), testNow, nil,
		)
	}

	tests := []struct {
		name    string
		status  user.RequestStatus
		wantErr bool
	}{
		{"pending transitions to used", user.RequestStatusPending, false},
		{"used returns error", user.RequestStatusUsed, true},
		{"expired returns error", user.RequestStatusExpired, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newReq(tt.status)
			err := req.MarkAsUsed(testNow)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if !tt.wantErr && req.Status() != user.RequestStatusUsed {
				t.Errorf("expected status USED, got %q", req.Status())
			}
		})
	}
}
