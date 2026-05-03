package user_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/user"
)

func TestNewEmail_Valid(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple email", "user@example.com", "user@example.com"},
		{"uppercase normalized", "User@Example.COM", "user@example.com"},
		{"with spaces trimmed", "  user@example.com  ", "user@example.com"},
		{"display name stripped", "john doe <john@example.com>", "john@example.com"},
		{"display name normalized", "jane doe <Jane@EXAMPLE.COM>", "jane@example.com"},
		{"unicode local part accepted", "ñoño@example.com", "ñoño@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email, err := user.NewEmail(tt.input)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if email.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, email.String())
			}
		})
	}
}

func TestNewEmail_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"only spaces", "   "},
		{"no at sign", "userexample.com"},
		{"no domain", "user@"},
		{"no local part", "@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := user.NewEmail(tt.input)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}
