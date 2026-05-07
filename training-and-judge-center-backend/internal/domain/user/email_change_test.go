package user_test

import (
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/user"
)

func TestEmailChangeRequest_MarkAsUsed(t *testing.T) {
	newReq := func(status user.RequestStatus) *user.EmailChangeRequest {
		email, _ := user.NewEmail("new@example.com")
		return user.RestoreEmailChangeRequest(
			"req-id", "user-id", email, "code123",
			status, testNow.Add(time.Hour), testNow, nil,
		)
	}

	tests := []struct {
		name    string
		status  user.RequestStatus
		wantErr bool
	}{
		{"pending transitions to used", user.StatusPending, false},
		{"used returns error", user.StatusUsed, true},
		{"expired returns error", user.StatusExpired, true},
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
			if !tt.wantErr && req.Status() != user.StatusUsed {
				t.Errorf("expected status USED, got %q", req.Status())
			}
		})
	}
}
