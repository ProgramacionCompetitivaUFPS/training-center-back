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
		{"PENDING", user.StatusPending},
		{"USED", user.StatusUsed},
		{"EXPIRED", user.StatusExpired},
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
