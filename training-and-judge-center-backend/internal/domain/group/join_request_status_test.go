package group_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/group"
)

func TestNewJoinRequestStatus_Valid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"pending", "PENDING"},
		{"approved", "APPROVED"},
		{"rejected", "REJECTED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := group.NewJoinRequestStatus(tt.input)
			if err != nil {
				t.Fatalf("expected no error for %q, got %v", tt.input, err)
			}
			if s.String() != tt.input {
				t.Errorf("String() = %q, want %q", s.String(), tt.input)
			}
		})
	}
}

func TestNewJoinRequestStatus_Invalid(t *testing.T) {
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
			_, err := group.NewJoinRequestStatus(tt.input)
			if err == nil {
				t.Errorf("expected error for input %q, got nil", tt.input)
			}
		})
	}
}

func TestJoinRequestStatus_StringAndRestore(t *testing.T) {
	if group.JoinRequestStatusApproved.String() != "APPROVED" {
		t.Errorf("String() = %q, want APPROVED", group.JoinRequestStatusApproved.String())
	}
	if got := group.RestoreJoinRequestStatus("LEGACY").String(); got != "LEGACY" {
		t.Errorf("RestoreJoinRequestStatus().String() = %q, want LEGACY: no validation on restore", got)
	}
}
