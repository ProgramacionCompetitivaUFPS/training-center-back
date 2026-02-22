package user

import "fmt"

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
		return "", fmt.Errorf("invalid status: %s", value)
	}
}

func (s Status) String() string {
	return string(s)
}
