package problem

import "github.com/training-judge-center/backend/pkg/apperror"

const (
	problemValidationStatusPending     = "PENDING"
	problemValidationStatusRunning     = "RUNNING"
	problemValidationStatusPassed      = "PASSED"
	problemValidationStatusFailed      = "FAILED"
	problemValidationStatusSystemError = "SYSTEM_ERROR"
)

type ProblemValidationStatus struct {
	value string
}

func NewProblemValidationStatus(raw string) (ProblemValidationStatus, error) {
	switch raw {
	case problemValidationStatusPending, problemValidationStatusRunning,
		problemValidationStatusPassed, problemValidationStatusFailed, problemValidationStatusSystemError:
		return ProblemValidationStatus{value: raw}, nil
	default:
		return ProblemValidationStatus{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "status", Message: "invalid status value"},
		})
	}
}

func RestoreProblemValidationStatus(value string) ProblemValidationStatus {
	return ProblemValidationStatus{value: value}
}

func NewProblemValidationStatusPending() ProblemValidationStatus {
	return ProblemValidationStatus{value: problemValidationStatusPending}
}
func NewProblemValidationStatusRunning() ProblemValidationStatus {
	return ProblemValidationStatus{value: problemValidationStatusRunning}
}
func NewProblemValidationStatusPassed() ProblemValidationStatus {
	return ProblemValidationStatus{value: problemValidationStatusPassed}
}
func NewProblemValidationStatusFailed() ProblemValidationStatus {
	return ProblemValidationStatus{value: problemValidationStatusFailed}
}
func NewProblemValidationStatusSystemError() ProblemValidationStatus {
	return ProblemValidationStatus{value: problemValidationStatusSystemError}
}

func (s ProblemValidationStatus) String() string { return s.value }

func (s ProblemValidationStatus) IsPending() bool { return s.value == problemValidationStatusPending }
func (s ProblemValidationStatus) IsRunning() bool { return s.value == problemValidationStatusRunning }
func (s ProblemValidationStatus) IsPassed() bool  { return s.value == problemValidationStatusPassed }
func (s ProblemValidationStatus) IsSystemError() bool {
	return s.value == problemValidationStatusSystemError
}
func (s ProblemValidationStatus) IsFinal() bool {
	switch s.value {
	case problemValidationStatusPassed, problemValidationStatusFailed, problemValidationStatusSystemError:
		return true
	}
	return false
}
