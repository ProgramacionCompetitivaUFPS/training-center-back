package problem_test

import (
	"reflect"
	"testing"

	"github.com/training-judge-center/backend/internal/domain/problem"
)

func TestNewLanguageLimit_KeepsTheValues(t *testing.T) {
	limit, err := problem.NewLanguageLimit("cpp20", 2000, 512)

	if err != nil {
		t.Fatalf("NewLanguageLimit: %v", err)
	}
	if limit.Language() != "cpp20" || limit.MaxTimeLimit() != 2000 || limit.MaxMemoryLimit() != 512 {
		t.Errorf("limit = %s/%d/%d, want cpp20/2000/512", limit.Language(), limit.MaxTimeLimit(), limit.MaxMemoryLimit())
	}
}

func TestNewLanguageLimit_Rejections(t *testing.T) {
	tests := []struct {
		name       string
		language   string
		time, mem  int
		wantFields []string
	}{
		{"an empty language", "", 2000, 512, []string{"language"}},
		{"a time limit of zero", "cpp20", 0, 512, []string{"maxTimeLimit"}},
		{"a negative memory limit", "cpp20", 2000, -1, []string{"maxMemoryLimit"}},
		{"everything wrong reports all three, in order", "", 0, 0, []string{"language", "maxTimeLimit", "maxMemoryLimit"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := problem.NewLanguageLimit(tt.language, tt.time, tt.mem)

			if got := detailFields(t, err); !reflect.DeepEqual(got, tt.wantFields) {
				t.Errorf("fields = %v, want %v", got, tt.wantFields)
			}
		})
	}
}
