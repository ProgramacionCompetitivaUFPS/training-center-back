package user

import "github.com/training-judge-center/backend/pkg/apperror"

const (
	requestStatusPending = "PENDING"
	requestStatusUsed    = "USED"
	requestStatusExpired = "EXPIRED"
)

type RequestStatus struct{ value string }

var (
	RequestStatusPending = RequestStatus{value: requestStatusPending}
	RequestStatusUsed    = RequestStatus{value: requestStatusUsed}
	RequestStatusExpired = RequestStatus{value: requestStatusExpired}
)

func NewRequestStatus(s string) (RequestStatus, error) {
	switch s {
	case requestStatusPending, requestStatusUsed, requestStatusExpired:
		return RequestStatus{value: s}, nil
	}
	return RequestStatus{}, apperror.NewValidation([]apperror.FieldError{
		{Field: "status", Message: "invalid request status: " + s},
	})
}

func RestoreRequestStatus(s string) RequestStatus { return RequestStatus{value: s} }

func (r RequestStatus) String() string { return r.value }

