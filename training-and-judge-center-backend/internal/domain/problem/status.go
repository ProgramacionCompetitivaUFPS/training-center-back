package problem

import "github.com/training-judge-center/backend/pkg/apperror"

type Status struct {
	value string
}

func NewStatus(value string) (Status, error) {
	switch value {
	case "DRAFT", "PUBLISHED":
		return Status{value: value}, nil
	default:
		return Status{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "status", Message: "Invalid status. Must be DRAFT or PUBLISHED"},
		})
	}
}

func NewStatusDraft() Status {
	return Status{value: "DRAFT"}
}

func NewStatusPublished() Status {
	return Status{value: "PUBLISHED"}
}

func (s Status) String() string {
	return s.value
}
