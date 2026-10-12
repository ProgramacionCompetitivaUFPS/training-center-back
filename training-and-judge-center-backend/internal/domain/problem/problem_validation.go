package problem

import (
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ProblemValidation struct {
	id            string
	problemID     string
	requestedBy   shared.UserID
	status        ProblemValidationStatus
	requestedAt   time.Time
	completedAt   *time.Time
	resultJSON    *string
	pendingChange *PendingJudgingChange
}

func (v *ProblemValidation) ID() string                           { return v.id }
func (v *ProblemValidation) ProblemID() string                    { return v.problemID }
func (v *ProblemValidation) RequestedBy() shared.UserID           { return v.requestedBy }
func (v *ProblemValidation) Status() ProblemValidationStatus      { return v.status }
func (v *ProblemValidation) RequestedAt() time.Time               { return v.requestedAt }
func (v *ProblemValidation) CompletedAt() *time.Time              { return v.completedAt }
func (v *ProblemValidation) ResultJSON() *string                  { return v.resultJSON }
func (v *ProblemValidation) PendingChange() *PendingJudgingChange { return v.pendingChange }

func (v *ProblemValidation) IsJudgingUpdate() bool { return v.pendingChange != nil }

func NewProblemValidation(id, problemID string, requestedBy shared.UserID, now time.Time) (*ProblemValidation, error) {
	if id == "" || problemID == "" {
		return nil, apperror.NewInternal()
	}
	return &ProblemValidation{
		id:          id,
		problemID:   problemID,
		requestedBy: requestedBy,
		status:      NewProblemValidationStatusPending(),
		requestedAt: now.UTC(),
	}, nil
}

func RestoreProblemValidation(
	id, problemID string,
	requestedBy shared.UserID,
	status ProblemValidationStatus,
	requestedAt time.Time,
	completedAt *time.Time,
	resultJSON *string,
	pendingChange *PendingJudgingChange,
) *ProblemValidation {
	return &ProblemValidation{
		id:            id,
		problemID:     problemID,
		requestedBy:   requestedBy,
		status:        status,
		requestedAt:   requestedAt,
		completedAt:   completedAt,
		resultJSON:    resultJSON,
		pendingChange: pendingChange,
	}
}

// Start transitions PENDING → RUNNING.
func (v *ProblemValidation) Start(now time.Time) error {
	if !v.status.IsPending() {
		return apperror.NewInternal()
	}
	v.status = NewProblemValidationStatusRunning()
	return nil
}

// markFinal is the shared guard for all RUNNING → final transitions.
func (v *ProblemValidation) markFinal(resultJSON string, now time.Time) error {
	if !v.status.IsRunning() {
		return apperror.NewInternal()
	}
	t := now.UTC()
	v.completedAt = &t
	v.resultJSON = &resultJSON
	return nil
}

func (v *ProblemValidation) MarkPassed(resultJSON string, now time.Time) error {
	if err := v.markFinal(resultJSON, now); err != nil {
		return err
	}
	v.status = NewProblemValidationStatusPassed()
	return nil
}

func (v *ProblemValidation) MarkFailed(resultJSON string, now time.Time) error {
	if err := v.markFinal(resultJSON, now); err != nil {
		return err
	}
	v.status = NewProblemValidationStatusFailed()
	return nil
}

func (v *ProblemValidation) MarkSystemError(resultJSON string, now time.Time) error {
	if err := v.markFinal(resultJSON, now); err != nil {
		return err
	}
	v.status = NewProblemValidationStatusSystemError()
	return nil
}

// NewJudgingUpdateValidation opens the ticket that validates a replacement of a published problem's judging files.
func NewJudgingUpdateValidation(id, problemID string, requestedBy shared.UserID, change PendingJudgingChange, now time.Time) (*ProblemValidation, error) {
	v, err := NewProblemValidation(id, problemID, requestedBy, now)
	if err != nil {
		return nil, err
	}
	v.pendingChange = &change
	return v, nil
}
