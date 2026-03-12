package problem_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/problem"
)

func TestNewTimeLimit(t *testing.T) {
	maxGlobal := 3000

	tests := []struct {
		name      string
		value     int
		maxGlobal int
		wantErr   bool
	}{
		{
			name:      "Valid time limit",
			value:     1000,
			maxGlobal: maxGlobal,
			wantErr:   false,
		},
		{
			name:      "Valid time limit at max",
			value:     3000,
			maxGlobal: maxGlobal,
			wantErr:   false,
		},
		{
			name:      "Zero time limit",
			value:     0,
			maxGlobal: maxGlobal,
			wantErr:   true,
		},
		{
			name:      "Negative time limit",
			value:     -100,
			maxGlobal: maxGlobal,
			wantErr:   true,
		},
		{
			name:      "Exceeds global max",
			value:     3001,
			maxGlobal: maxGlobal,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tl, err := problem.NewTimeLimit(tt.value, tt.maxGlobal)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
					return
				}

				if tl.Value() != tt.value {
					t.Errorf("expected value %d, got %d", tt.value, tl.Value())
				}
			}
		})
	}
}
