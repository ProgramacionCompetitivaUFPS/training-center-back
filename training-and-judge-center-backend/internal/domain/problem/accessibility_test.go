package problem_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/problem"
)

func TestNewAccessibility(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"private is valid", "PRIVATE", false},
		{"public is valid", "PUBLIC", false},
		{"invalid string returns error", "INVALID", true},
		{"empty string returns error", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := problem.NewAccessibility(tt.input)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestAccessibilityFactories(t *testing.T) {
	if got := problem.NewAccessibilityPrivate().String(); got != "PRIVATE" {
		t.Errorf("NewAccessibilityPrivate: got %q, want PRIVATE", got)
	}
	if got := problem.NewAccessibilityPublic().String(); got != "PUBLIC" {
		t.Errorf("NewAccessibilityPublic: got %q, want PUBLIC", got)
	}
}
