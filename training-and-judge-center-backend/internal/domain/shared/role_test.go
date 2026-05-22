package shared_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/shared"
)

func TestNewRole_Valid(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected shared.Role
	}{
		{"admin role", "ADMIN", shared.RoleAdmin},
		{"coach role", "COACH", shared.RoleCoach},
		{"contestant role", "CONTESTANT", shared.RoleContestant},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role, err := shared.NewRole(tt.input)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if role != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, role)
			}
		})
	}
}

func TestNewRole_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"lowercase admin", "admin"},
		{"unknown role", "SUPERUSER"},
		{"partial match", "ADMINN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := shared.NewRole(tt.input)
			if err == nil {
				t.Errorf("expected error for input %q, got nil", tt.input)
			}
		})
	}
}

func TestRole_IsValid(t *testing.T) {
	var zeroRole shared.Role
	tests := []struct {
		name     string
		role     shared.Role
		expected bool
	}{
		{"valid admin", shared.RoleAdmin, true},
		{"valid coach", shared.RoleCoach, true},
		{"valid contestant", shared.RoleContestant, true},
		{"zero value", zeroRole, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.role.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}
