package problem_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/problem"
)

func TestNewMemoryLimit(t *testing.T) {
	maxGlobal := 512

	tests := []struct {
		name      string
		value     int
		maxGlobal int
		wantErr   bool
	}{
		{
			name:      "Valid memory limit",
			value:     256,
			maxGlobal: maxGlobal,
			wantErr:   false,
		},
		{
			name:      "Valid memory limit at max",
			value:     512,
			maxGlobal: maxGlobal,
			wantErr:   false,
		},
		{
			name:      "Zero memory limit",
			value:     0,
			maxGlobal: maxGlobal,
			wantErr:   true,
		},
		{
			name:      "Negative memory limit",
			value:     -256,
			maxGlobal: maxGlobal,
			wantErr:   true,
		},
		{
			name:      "Exceeds global max",
			value:     513,
			maxGlobal: maxGlobal,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ml, err := problem.NewMemoryLimit(tt.value, tt.maxGlobal)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
					return
				}

				if ml.Value() != tt.value {
					t.Errorf("expected value %d, got %d", tt.value, ml.Value())
				}
			}
		})
	}
}
