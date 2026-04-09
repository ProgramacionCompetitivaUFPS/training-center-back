package user

import "testing"

func TestNewStatus_Valid(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Status
	}{
		{"active status", "ACTIVE", StatusActive},
		{"deactivated status", "DEACTIVATED", StatusDeactivated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := NewStatus(tt.input)
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
			_, err := NewStatus(tt.input)
			if err == nil {
				t.Errorf("expected error for input %q, got nil", tt.input)
			}
		})
	}
}

func TestStatus_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected bool
	}{
		{"valid active", StatusActive, true},
		{"valid deactivated", StatusDeactivated, true},
		{"arbitrary cast bypass", Status("SUSPENDED"), false},
		{"empty status", Status(""), false},
		{"lowercase bypass", Status("active"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}
