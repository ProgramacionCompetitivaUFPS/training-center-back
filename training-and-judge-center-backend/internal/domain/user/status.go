package user

import "github.com/training-judge-center/backend/pkg/apperror"

type Status string

const (
	StatusActive      Status = "ACTIVE"
	StatusDeactivated Status = "DEACTIVATED"
)

func NewStatus(value string) (Status, error) {
	switch Status(value) {
	case StatusActive, StatusDeactivated:
		return Status(value), nil
	default:
		return "", apperror.NewValidation([]apperror.FieldError{
			{Field: "status", Message: "invalid status: " + value},
		})
	}
}

func RestoreStatus(value string) Status {
	return Status(value)
}

func (s Status) IsValid() bool {
	switch s {
	case StatusActive, StatusDeactivated:
		return true
	}
	return false
}

func (s Status) String() string {
	return string(s)
}
