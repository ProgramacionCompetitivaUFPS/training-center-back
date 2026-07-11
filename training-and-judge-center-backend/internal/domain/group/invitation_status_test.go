package group_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/group"
)

func TestNewInvitationStatus_Valid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"pending", "PENDING"},
		{"accepted", "ACCEPTED"},
		{"revoked", "REVOKED"},
		{"expired", "EXPIRED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := group.NewInvitationStatus(tt.input)
			if err != nil {
				t.Fatalf("expected no error for %q, got %v", tt.input, err)
			}
			if s.String() != tt.input {
				t.Errorf("String() = %q, want %q", s.String(), tt.input)
			}
		})
	}
}

func TestNewInvitationStatus_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"lowercase", "pending"},
		{"unknown value", "INVALID"},
		{"cancelled", "CANCELLED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := group.NewInvitationStatus(tt.input)
			if err == nil {
				t.Errorf("expected error for input %q, got nil", tt.input)
			}
		})
	}
}
