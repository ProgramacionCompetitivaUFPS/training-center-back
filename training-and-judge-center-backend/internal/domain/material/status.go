package material

import "github.com/training-judge-center/backend/pkg/apperror"

const (
	statusDraft     = "DRAFT"
	statusPublished = "PUBLISHED"
)

type Status struct {
	value string
}

func NewStatus(value string) (Status, error) {
	switch value {
	case statusDraft, statusPublished:
		return Status{value: value}, nil
	default:
		return Status{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "status", Message: "invalid status: must be DRAFT or PUBLISHED"},
		})
	}
}

func NewStatusDraft() Status            { return Status{value: statusDraft} }
func NewStatusPublished() Status        { return Status{value: statusPublished} }
func RestoreStatus(value string) Status { return Status{value: value} }

func (s Status) String() string    { return s.value }
func (s Status) IsPublished() bool { return s.value == statusPublished }
func (s Status) IsDraft() bool     { return s.value == statusDraft }
