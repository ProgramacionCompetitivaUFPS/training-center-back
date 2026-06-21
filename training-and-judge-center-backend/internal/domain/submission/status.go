package submission

import "github.com/training-judge-center/backend/pkg/apperror"

const (
	statusPending             = "PENDING"
	statusRunning             = "RUNNING"
	statusAccepted            = "ACCEPTED"
	statusWrongAnswer         = "WRONG_ANSWER"
	statusTimeLimitExceeded   = "TIME_LIMIT_EXCEEDED"
	statusMemoryLimitExceeded = "MEMORY_LIMIT_EXCEEDED"
	statusRuntimeError        = "RUNTIME_ERROR"
	statusCompilationError    = "COMPILATION_ERROR"
	statusSystemError         = "SYSTEM_ERROR"
)

type SubmissionStatus struct {
	value string
}

func NewStatus(raw string) (SubmissionStatus, error) {
	switch raw {
	case statusPending, statusRunning, statusAccepted, statusWrongAnswer,
		statusTimeLimitExceeded, statusMemoryLimitExceeded,
		statusRuntimeError, statusCompilationError, statusSystemError:
		return SubmissionStatus{value: raw}, nil
	default:
		return SubmissionStatus{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "status", Message: "invalid status value"},
		})
	}
}

func RestoreStatus(value string) SubmissionStatus {
	return SubmissionStatus{value: value}
}

func newStatusPending() SubmissionStatus             { return SubmissionStatus{value: statusPending} }
func newStatusRunning() SubmissionStatus             { return SubmissionStatus{value: statusRunning} }
func newStatusAccepted() SubmissionStatus            { return SubmissionStatus{value: statusAccepted} }
func newStatusWrongAnswer() SubmissionStatus         { return SubmissionStatus{value: statusWrongAnswer} }
func newStatusTimeLimitExceeded() SubmissionStatus   { return SubmissionStatus{value: statusTimeLimitExceeded} }
func newStatusMemoryLimitExceeded() SubmissionStatus { return SubmissionStatus{value: statusMemoryLimitExceeded} }
func newStatusRuntimeError() SubmissionStatus        { return SubmissionStatus{value: statusRuntimeError} }
func newStatusCompilationError() SubmissionStatus    { return SubmissionStatus{value: statusCompilationError} }
func newStatusSystemError() SubmissionStatus         { return SubmissionStatus{value: statusSystemError} }

func (s SubmissionStatus) String() string { return s.value }

func (s SubmissionStatus) IsPending() bool { return s.value == statusPending }
func (s SubmissionStatus) IsRunning() bool { return s.value == statusRunning }
func (s SubmissionStatus) IsFinal() bool {
	switch s.value {
	case statusAccepted, statusWrongAnswer, statusTimeLimitExceeded,
		statusMemoryLimitExceeded, statusRuntimeError, statusCompilationError, statusSystemError:
		return true
	}
	return false
}
