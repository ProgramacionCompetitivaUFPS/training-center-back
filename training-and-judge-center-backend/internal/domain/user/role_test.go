package user

import "testing"

func TestNewRole_Valid(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Role
	}{
		{"admin role", "ADMIN", RoleAdmin},
		{"coach role", "COACH", RoleCoach},
		{"contestant role", "CONTESTANT", RoleContestant},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role, err := NewRole(tt.input)
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
			_, err := NewRole(tt.input)
			if err == nil {
				t.Errorf("expected error for input %q, got nil", tt.input)
			}
		})
	}
}

func TestRole_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		role     Role
		expected bool
	}{
		{"valid admin", RoleAdmin, true},
		{"valid coach", RoleCoach, true},
		{"valid contestant", RoleContestant, true},
		{"arbitrary cast bypass", Role("SUPERUSER"), false},
		{"empty role", Role(""), false},
		{"lowercase bypass", Role("admin"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.role.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}
