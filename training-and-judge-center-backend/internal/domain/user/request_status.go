package user

import "github.com/training-judge-center/backend/pkg/apperror"

type RequestStatus string

const (
	StatusPending RequestStatus = "PENDING"
	StatusUsed    RequestStatus = "USED"
	StatusExpired RequestStatus = "EXPIRED"
)

func NewRequestStatus(s string) (RequestStatus, error) {
	switch RequestStatus(s) {
	case StatusPending, StatusUsed, StatusExpired:
		return RequestStatus(s), nil
	}
	return "", apperror.NewValidation([]apperror.FieldError{
		{Field: "status", Message: "invalid request status: " + s},
	})
}

