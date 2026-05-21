package submission_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/submission"
)

func TestNewLanguage(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"cpp20 is valid", "cpp20", false},
		{"java17 is valid", "java17", false},
		{"python310 is valid", "python310", false},
		{"uppercase returns error", "CPP20", true},
		{"empty string returns error", "", true},
		{"unknown language returns error", "go", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := submission.NewLanguage(tt.input)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestLanguage_String(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"cpp20", "cpp20"},
		{"java17", "java17"},
		{"python310", "python310"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lang, err := submission.NewLanguage(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := lang.String(); got != tt.want {
				t.Errorf("String(): got %q, want %q", got, tt.want)
			}
		})
	}
}
