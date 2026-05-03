package user

import "github.com/training-judge-center/backend/pkg/apperror"

type Status struct{ value string }

var (
	StatusActive      = Status{value: "ACTIVE"}
	StatusDeactivated = Status{value: "DEACTIVATED"}
)

func NewStatus(value string) (Status, error) {
	switch value {
	case "ACTIVE", "DEACTIVATED":
		return Status{value: value}, nil
	default:
		return Status{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "status", Message: "invalid status: " + value},
		})
	}
}

func RestoreStatus(value string) Status {
	return Status{value: value}
}

func (s Status) IsValid() bool {
	switch s {
	case StatusActive, StatusDeactivated:
		return true
	}
	return false
}

func (s Status) String() string {
	return s.value
}
