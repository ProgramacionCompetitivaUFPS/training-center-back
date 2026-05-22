package user_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/user"
)

func TestNewStatus_Valid(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected user.Status
	}{
		{"active status", "ACTIVE", user.StatusActive},
		{"deactivated status", "DEACTIVATED", user.StatusDeactivated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := user.NewStatus(tt.input)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if status != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, status)
			}
		})
	}
}

func TestNewStatus_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"lowercase active", "active"},
		{"unknown status", "SUSPENDED"},
		{"partial match", "ACTIVEE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := user.NewStatus(tt.input)
			if err == nil {
				t.Errorf("expected error for input %q, got nil", tt.input)
			}
		})
	}
}

func TestStatus_IsValid(t *testing.T) {
	var zeroStatus user.Status
	tests := []struct {
		name     string
		status   user.Status
		expected bool
	}{
		{"valid active", user.StatusActive, true},
		{"valid deactivated", user.StatusDeactivated, true},
		{"zero value", zeroStatus, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}
