package problem_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/problem"
)

func TestNewLanguageOverride(t *testing.T) {
	globalMaxTime := 3000
	globalMaxMemory := 512

	cppLimitVal, err := problem.NewLanguageLimit("cpp20", 1000, 256)
	if err != nil {
		t.Fatal(err)
	}
	cppLimit := &cppLimitVal

	timePtr := func(v int) *int { return &v }

	tests := []struct {
		name            string
		language        string
		timeLimit       *int
		memoryLimit     *int
		langLimit       *problem.LanguageLimit
		globalMaxTime   int
		globalMaxMemory int
		wantErr         bool
	}{
		{
			name:            "Valid limits for known language",
			language:        "cpp20",
			timeLimit:       timePtr(1000),
			memoryLimit:     timePtr(256),
			langLimit:       cppLimit,
			globalMaxTime:   globalMaxTime,
			globalMaxMemory: globalMaxMemory,
			wantErr:         false,
		},
		{
			name:            "Time limit exceeds known language max",
			language:        "cpp20",
			timeLimit:       timePtr(1001),
			memoryLimit:     timePtr(256),
			langLimit:       cppLimit,
			globalMaxTime:   globalMaxTime,
			globalMaxMemory: globalMaxMemory,
			wantErr:         true,
		},
		{
			name:            "Valid limits for unknown language (fallback to globals)",
			language:        "rust1.75",
			timeLimit:       timePtr(3000),
			memoryLimit:     timePtr(512),
			langLimit:       nil,
			globalMaxTime:   globalMaxTime,
			globalMaxMemory: globalMaxMemory,
			wantErr:         false,
		},
		{
			name:            "Time limit exceeds global max for unknown language",
			language:        "rust1.75",
			timeLimit:       timePtr(3001),
			memoryLimit:     timePtr(512),
			langLimit:       nil,
			globalMaxTime:   globalMaxTime,
			globalMaxMemory: globalMaxMemory,
			wantErr:         true,
		},
		{
			name:            "Empty language",
			language:        "",
			timeLimit:       timePtr(1000),
			memoryLimit:     timePtr(256),
			langLimit:       nil,
			globalMaxTime:   globalMaxTime,
			globalMaxMemory: globalMaxMemory,
			wantErr:         true,
		},
		{
			name:            "Negative limits",
			language:        "cpp20",
			timeLimit:       timePtr(-100),
			memoryLimit:     timePtr(-256),
			langLimit:       cppLimit,
			globalMaxTime:   globalMaxTime,
			globalMaxMemory: globalMaxMemory,
			wantErr:         true,
		},
		{
			name:            "Nil limits (valid, means inherit default)",
			language:        "cpp20",
			timeLimit:       nil,
			memoryLimit:     nil,
			langLimit:       cppLimit,
			globalMaxTime:   globalMaxTime,
			globalMaxMemory: globalMaxMemory,
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := problem.NewLanguageOverride(
				tt.language,
				tt.timeLimit,
				tt.memoryLimit,
				tt.langLimit,
				tt.globalMaxTime,
				tt.globalMaxMemory,
			)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}
