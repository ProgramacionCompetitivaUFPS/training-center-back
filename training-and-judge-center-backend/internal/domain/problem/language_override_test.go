package problem_test

import (
	"reflect"
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

func TestNewLanguageOverride_KeepsTheValues(t *testing.T) {
	limit, err := problem.NewLanguageLimit("cpp20", 1000, 256)
	if err != nil {
		t.Fatal(err)
	}

	lo, err := problem.NewLanguageOverride("cpp20", intPtr(500), intPtr(128), &limit, 3000, 512)

	if err != nil {
		t.Fatalf("NewLanguageOverride: %v", err)
	}
	if lo.Language() != "cpp20" || lo.TimeLimit() == nil || *lo.TimeLimit() != 500 || lo.MemoryLimit() == nil || *lo.MemoryLimit() != 128 {
		t.Errorf("override = %s/%v/%v, want cpp20/500/128", lo.Language(), lo.TimeLimit(), lo.MemoryLimit())
	}
}

func TestNewLanguageOverride_Rejections(t *testing.T) {
	limit, err := problem.NewLanguageLimit("cpp20", 1000, 256)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		language   string
		time, mem  *int
		langLimit  *problem.LanguageLimit
		wantFields []string
	}{
		{"an empty language", "", nil, nil, &limit, []string{"languageOverrides.language"}},
		{"a time limit of zero", "cpp20", intPtr(0), nil, &limit, []string{"languageOverrides.timeLimit"}},
		{"a negative memory limit", "cpp20", nil, intPtr(-1), &limit, []string{"languageOverrides.memoryLimit"}},
		{"a time limit above the language maximum", "cpp20", intPtr(1001), nil, &limit, []string{"languageOverrides.timeLimit"}},
		{"a memory limit above the language maximum", "cpp20", nil, intPtr(257), &limit, []string{"languageOverrides.memoryLimit"}},
		{"a time limit above the global maximum for an unlisted language", "cobol", intPtr(3001), nil, nil, []string{"languageOverrides.timeLimit"}},
		{"a memory limit above the global maximum for an unlisted language", "cobol", nil, intPtr(513), nil, []string{"languageOverrides.memoryLimit"}},
		{"everything wrong reports every field, in order", "", intPtr(0), intPtr(0), nil, []string{"languageOverrides.language", "languageOverrides.timeLimit", "languageOverrides.memoryLimit"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := problem.NewLanguageOverride(tt.language, tt.time, tt.mem, tt.langLimit, 3000, 512)

			if got := detailFields(t, err); !reflect.DeepEqual(got, tt.wantFields) {
				t.Errorf("fields = %v, want %v", got, tt.wantFields)
			}
		})
	}
}

func TestNewLanguageOverride_AcceptsLimitsEqualToTheMaximum(t *testing.T) {
	limit, err := problem.NewLanguageLimit("cpp20", 1000, 256)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := problem.NewLanguageOverride("cpp20", intPtr(1000), intPtr(256), &limit, 3000, 512); err != nil {
		t.Errorf("limits equal to the language maximum: %v", err)
	}
	if _, err := problem.NewLanguageOverride("cobol", intPtr(3000), intPtr(512), nil, 3000, 512); err != nil {
		t.Errorf("limits equal to the global maximum: %v", err)
	}
}

func TestRestoreLanguageOverride_KeepsValuesWithoutValidating(t *testing.T) {
	lo := problem.RestoreLanguageOverride("", intPtr(-5), nil)

	if lo.Language() != "" || lo.TimeLimit() == nil || *lo.TimeLimit() != -5 || lo.MemoryLimit() != nil {
		t.Errorf("restored override = %q/%v/%v, want the stored values untouched", lo.Language(), lo.TimeLimit(), lo.MemoryLimit())
	}
}
