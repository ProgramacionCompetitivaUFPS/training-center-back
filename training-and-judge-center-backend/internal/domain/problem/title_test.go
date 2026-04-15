package problem_test

import (
	"strings"
	"testing"

	"github.com/training-judge-center/backend/internal/domain/problem"
)

func TestNewTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "Valid simple title",
			input:    "Sum of Two Numbers",
			expected: "Sum of Two Numbers",
			wantErr:  false,
		},
		{
			name:     "Valid title with surrounding spaces",
			input:    "  Sum of Two Numbers  ",
			expected: "Sum of Two Numbers",
			wantErr:  false,
		},
		{
			name:     "Title with unicode normalization needs",
			input:    "Café",
			expected: "Café",
			wantErr:  false,
		},
		{
			name:     "Empty title",
			input:    "",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Title with only spaces",
			input:    "    ",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Title too long (201 chars)",
			input:    strings.Repeat("a", 201),
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Title exactly at max length (200 chars)",
			input:    strings.Repeat("a", 200),
			expected: strings.Repeat("a", 200),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, err := problem.NewTitle(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
					return
				}

				if title.String() != tt.expected {
					t.Errorf("expected value %s, got %s", tt.expected, title.String())
				}
			}
		})
	}
}
