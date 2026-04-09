package user

import (
	"testing"
	"time"
)

func TestEmailChangeRequest_MarkAsUsed(t *testing.T) {
	now := time.Now()

	newReq := func(status RequestStatus) *EmailChangeRequest {
		email, _ := NewEmail("new@example.com")
		return RestoreEmailChangeRequest(
			"req-id", "user-id", email, "code123",
			status, now.Add(time.Hour), now, nil,
		)
	}

	tests := []struct {
		name      string
		status    RequestStatus
		wantErr   bool
	}{
		{"pending transitions to used", StatusPending, false},
		{"used returns error", StatusUsed, true},
		{"expired returns error", StatusExpired, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newReq(tt.status)
			err := req.MarkAsUsed(now)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if !tt.wantErr && req.Status() != StatusUsed {
				t.Errorf("expected status USED, got %q", req.Status())
			}
		})
	}
}
