package material_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/material"
)

func TestNewStatus_Valid(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantDraft     bool
		wantPublished bool
	}{
		{"DRAFT is valid", "DRAFT", true, false},
		{"PUBLISHED is valid", "PUBLISHED", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := material.NewStatus(tt.input)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if s.IsDraft() != tt.wantDraft {
				t.Errorf("IsDraft() = %v, want %v", s.IsDraft(), tt.wantDraft)
			}
			if s.IsPublished() != tt.wantPublished {
				t.Errorf("IsPublished() = %v, want %v", s.IsPublished(), tt.wantPublished)
			}
			if s.String() != tt.input {
				t.Errorf("String() = %q, want %q", s.String(), tt.input)
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
		{"lowercase draft", "draft"},
		{"lowercase published", "published"},
		{"unknown value", "PENDING"},
		{"partial match", "DRAFTX"},
		{"whitespace prefix", " DRAFT"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := material.NewStatus(tt.input)
			if err == nil {
				t.Errorf("expected validation error for input %q, got nil", tt.input)
			}
		})
	}
}

func TestStatusFactories(t *testing.T) {
	draft := material.NewStatusDraft()
	if !draft.IsDraft() {
		t.Error("NewStatusDraft() should return draft status")
	}
	if draft.IsPublished() {
		t.Error("NewStatusDraft() should not be published")
	}
	if draft.String() != "DRAFT" {
		t.Errorf("String() = %q, want DRAFT", draft.String())
	}

	published := material.NewStatusPublished()
	if !published.IsPublished() {
		t.Error("NewStatusPublished() should return published status")
	}
	if published.IsDraft() {
		t.Error("NewStatusPublished() should not be draft")
	}
	if published.String() != "PUBLISHED" {
		t.Errorf("String() = %q, want PUBLISHED", published.String())
	}
}

func TestRestoreStatus_DoesNotValidate(t *testing.T) {
	s := material.RestoreStatus("UNKNOWN_VALUE_FROM_DB")
	if s.String() != "UNKNOWN_VALUE_FROM_DB" {
		t.Errorf("RestoreStatus should preserve raw value, got %q", s.String())
	}
}
