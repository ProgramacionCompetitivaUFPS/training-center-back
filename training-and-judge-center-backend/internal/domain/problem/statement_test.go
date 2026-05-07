package problem_test

import (
	"strings"
	"testing"

	"github.com/training-judge-center/backend/internal/domain/problem"
)

func TestNewStatement(t *testing.T) {
	atLimit := strings.Repeat("a", 150_000)
	overLimit := strings.Repeat("a", 150_001)
	short := "Hello, world!"

	tests := []struct {
		name    string
		input   *string
		wantNil bool
		wantErr bool
	}{
		{"nil is valid", nil, true, false},
		{"short string is valid", &short, false, false},
		{"exactly 150_000 runes is valid", &atLimit, false, false},
		{"150_001 runes returns error", &overLimit, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := problem.NewStatement(tt.input)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if !tt.wantErr {
				got := s.Value()
				if tt.wantNil && got != nil {
					t.Errorf("Value(): got %q, want nil", *got)
				}
				if !tt.wantNil && got == nil {
					t.Error("Value(): got nil, want non-nil")
				}
			}
		})
	}
}
