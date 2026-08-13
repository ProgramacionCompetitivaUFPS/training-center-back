package problem_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/problem"
)

func TestNewProblemValidationStatus(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"pending is valid", "PENDING", false},
		{"running is valid", "RUNNING", false},
		{"passed is valid", "PASSED", false},
		{"failed is valid", "FAILED", false},
		{"system error is valid", "SYSTEM_ERROR", false},
		{"invalid string returns error", "INVALID", true},
		{"empty string returns error", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := problem.NewProblemValidationStatus(tt.input)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestProblemValidationStatus_IsPending(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"pending is pending", "PENDING", true},
		{"running is not pending", "RUNNING", false},
		{"passed is not pending", "PASSED", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := problem.NewProblemValidationStatus(tt.input)
			if got := s.IsPending(); got != tt.want {
				t.Errorf("IsPending(): got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProblemValidationStatus_IsRunning(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"running is running", "RUNNING", true},
		{"pending is not running", "PENDING", false},
		{"passed is not running", "PASSED", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := problem.NewProblemValidationStatus(tt.input)
			if got := s.IsRunning(); got != tt.want {
				t.Errorf("IsRunning(): got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProblemValidationStatus_IsPassed(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"passed is passed", "PASSED", true},
		{"failed is not passed", "FAILED", false},
		{"system error is not passed", "SYSTEM_ERROR", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := problem.NewProblemValidationStatus(tt.input)
			if got := s.IsPassed(); got != tt.want {
				t.Errorf("IsPassed(): got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProblemValidationStatus_IsSystemError(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"system error is system error", "SYSTEM_ERROR", true},
		{"failed is not system error", "FAILED", false},
		{"passed is not system error", "PASSED", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := problem.NewProblemValidationStatus(tt.input)
			if got := s.IsSystemError(); got != tt.want {
				t.Errorf("IsSystemError(): got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProblemValidationStatus_IsFinal(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"pending is not final", "PENDING", false},
		{"running is not final", "RUNNING", false},
		{"passed is final", "PASSED", true},
		{"failed is final", "FAILED", true},
		{"system error is final", "SYSTEM_ERROR", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := problem.NewProblemValidationStatus(tt.input)
			if got := s.IsFinal(); got != tt.want {
				t.Errorf("IsFinal(): got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProblemValidationStatusFactories(t *testing.T) {
	if got := problem.NewProblemValidationStatusPending().String(); got != "PENDING" {
		t.Errorf("NewProblemValidationStatusPending: got %q, want PENDING", got)
	}
	if got := problem.NewProblemValidationStatusRunning().String(); got != "RUNNING" {
		t.Errorf("NewProblemValidationStatusRunning: got %q, want RUNNING", got)
	}
	if got := problem.NewProblemValidationStatusPassed().String(); got != "PASSED" {
		t.Errorf("NewProblemValidationStatusPassed: got %q, want PASSED", got)
	}
	if got := problem.NewProblemValidationStatusFailed().String(); got != "FAILED" {
		t.Errorf("NewProblemValidationStatusFailed: got %q, want FAILED", got)
	}
	if got := problem.NewProblemValidationStatusSystemError().String(); got != "SYSTEM_ERROR" {
		t.Errorf("NewProblemValidationStatusSystemError: got %q, want SYSTEM_ERROR", got)
	}
}

func TestRestoreProblemValidationStatus(t *testing.T) {
	s := problem.RestoreProblemValidationStatus("PASSED")
	if s.String() != "PASSED" {
		t.Errorf("String(): got %q, want PASSED", s.String())
	}
}
