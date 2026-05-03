package user

import "github.com/training-judge-center/backend/pkg/apperror"

type RequestStatus struct{ value string }

var (
	StatusPending = RequestStatus{value: "PENDING"}
	StatusUsed    = RequestStatus{value: "USED"}
	StatusExpired = RequestStatus{value: "EXPIRED"}
)

func NewRequestStatus(s string) (RequestStatus, error) {
	switch s {
	case "PENDING", "USED", "EXPIRED":
		return RequestStatus{value: s}, nil
	}
	return RequestStatus{}, apperror.NewValidation([]apperror.FieldError{
		{Field: "status", Message: "invalid request status: " + s},
	})
}

func RestoreRequestStatus(s string) RequestStatus { return RequestStatus{value: s} }

func (r RequestStatus) String() string { return r.value }

