package submission_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/submission"
)

func TestSubmissionStatus_Predicates(t *testing.T) {
	tests := []struct {
		status    string
		isPending bool
		isRunning bool
		isFinal   bool
	}{
		{"PENDING", true, false, false},
		{"RUNNING", false, true, false},
		{"ACCEPTED", false, false, true},
		{"WRONG_ANSWER", false, false, true},
		{"TIME_LIMIT_EXCEEDED", false, false, true},
		{"MEMORY_LIMIT_EXCEEDED", false, false, true},
		{"RUNTIME_ERROR", false, false, true},
		{"COMPILATION_ERROR", false, false, true},
		{"SYSTEM_ERROR", false, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			s := submission.RestoreStatus(tt.status)
			if got := s.IsPending(); got != tt.isPending {
				t.Errorf("IsPending(): got %v, want %v", got, tt.isPending)
			}
			if got := s.IsRunning(); got != tt.isRunning {
				t.Errorf("IsRunning(): got %v, want %v", got, tt.isRunning)
			}
			if got := s.IsFinal(); got != tt.isFinal {
				t.Errorf("IsFinal(): got %v, want %v", got, tt.isFinal)
			}
		})
	}
}

func TestSubmissionStatus_String(t *testing.T) {
	s := submission.RestoreStatus("ACCEPTED")
	if got := s.String(); got != "ACCEPTED" {
		t.Errorf("String(): got %q, want ACCEPTED", got)
	}
}
