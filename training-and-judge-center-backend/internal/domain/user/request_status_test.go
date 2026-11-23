package user_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/user"
)

func TestNewRequestStatus_Valid(t *testing.T) {
	tests := []struct {
		input    string
		expected user.RequestStatus
	}{
		{"PENDING", user.RequestStatusPending},
		{"USED", user.RequestStatusUsed},
		{"EXPIRED", user.RequestStatusExpired},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := user.NewRequestStatus(tt.input)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestNewRequestStatus_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"lowercase pending", "pending"},
		{"unknown value", "CANCELLED"},
		{"partial match", "PEND"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := user.NewRequestStatus(tt.input)
			if err == nil {
				t.Errorf("expected error for input %q, got nil", tt.input)
			}
		})
	}
}

func TestRequestStatus_String(t *testing.T) {
	tests := []struct {
		status user.RequestStatus
		want   string
	}{
		{user.RequestStatusPending, "PENDING"},
		{user.RequestStatusUsed, "USED"},
		{user.RequestStatusExpired, "EXPIRED"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRestoreRequestStatus_KeepsValueWithoutValidating(t *testing.T) {
	if got := user.RestoreRequestStatus("LEGACY").String(); got != "LEGACY" {
		t.Errorf("String() = %q, want LEGACY", got)
	}
}
