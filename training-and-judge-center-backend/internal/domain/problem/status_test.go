package problem_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/problem"
)

func TestNewStatus(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"draft is valid", "DRAFT", false},
		{"published is valid", "PUBLISHED", false},
		{"invalid string returns error", "INVALID", true},
		{"empty string returns error", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := problem.NewStatus(tt.input)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStatus_IsPublished(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"draft is not published", "DRAFT", false},
		{"published is published", "PUBLISHED", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := problem.NewStatus(tt.input)
			if got := s.IsPublished(); got != tt.want {
				t.Errorf("IsPublished(): got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatusFactories(t *testing.T) {
	if got := problem.NewStatusDraft().String(); got != "DRAFT" {
		t.Errorf("NewStatusDraft: got %q, want DRAFT", got)
	}
	if got := problem.NewStatusPublished().String(); got != "PUBLISHED" {
		t.Errorf("NewStatusPublished: got %q, want PUBLISHED", got)
	}
}
