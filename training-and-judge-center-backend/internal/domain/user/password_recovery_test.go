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

func TestNewPasswordRecoveryRequest_StartsPendingAndExpiresInFifteenMinutes(t *testing.T) {
	req, err := user.NewPasswordRecoveryRequest("req-1", "user-1", "123456", localNow)

	if err != nil {
		t.Fatalf("NewPasswordRecoveryRequest: %v", err)
	}
	if req.ID() != "req-1" || req.UserID() != "user-1" || req.Code() != "123456" {
		t.Errorf("fields = %s/%s/%s, want req-1/user-1/123456", req.ID(), req.UserID(), req.Code())
	}
	if req.Status() != user.RequestStatusPending || req.UpdatedAt() != nil {
		t.Errorf("state = %s/%v, want PENDING and never updated", req.Status(), req.UpdatedAt())
	}
	assertInUTC(t, "CreatedAt", req.CreatedAt(), testNow)
	assertInUTC(t, "ExpiresAt", req.ExpiresAt(), time.Date(2024, 1, 15, 12, 15, 0, 0, time.UTC))
}

func TestNewPasswordRecoveryRequest_MissingData_IsAnInternalError(t *testing.T) {
	tests := []struct {
		name             string
		id, userID, code string
	}{
		{"an empty id", "", "user-1", "123456"},
		{"an empty user id", "req-1", "", "123456"},
		{"an empty code", "req-1", "user-1", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := user.NewPasswordRecoveryRequest(tt.id, tt.userID, tt.code, testNow)

			assertInternalError(t, tt.name, err)
			if req != nil {
				t.Errorf("expected no request, got %+v", req)
			}
		})
	}
}

func TestPasswordRecoveryRequest_IsExpired(t *testing.T) {
	expiresAt := testNow.Add(15 * time.Minute)
	tests := []struct {
		name   string
		now    time.Time
		status user.RequestStatus
		want   bool
	}{
		{"well before the expiry", testNow, user.RequestStatusPending, false},
		{"well after the expiry", expiresAt.Add(time.Minute), user.RequestStatusPending, true},
		{"marked as expired even before the time", testNow, user.RequestStatusExpired, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := user.RestorePasswordRecoveryRequest("req-1", "user-1", "123456", tt.status, expiresAt, testNow, nil)

			if got := req.IsExpired(tt.now); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRestorePasswordRecoveryRequest_KeepsTheStoredValues(t *testing.T) {
	updatedAt := testNow.Add(time.Minute)

	req := user.RestorePasswordRecoveryRequest("req-1", "user-1", "123456", user.RequestStatusUsed, testNow.Add(time.Hour), testNow, &updatedAt)

	if req.Status() != user.RequestStatusUsed || !req.ExpiresAt().Equal(testNow.Add(time.Hour)) || !req.CreatedAt().Equal(testNow) || req.UpdatedAt() == nil || !req.UpdatedAt().Equal(updatedAt) {
		t.Errorf("restored request = %s/%v/%v/%v", req.Status(), req.ExpiresAt(), req.CreatedAt(), req.UpdatedAt())
	}
}
