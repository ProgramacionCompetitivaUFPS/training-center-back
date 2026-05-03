package group

import "github.com/training-judge-center/backend/pkg/apperror"

type JoinRequestStatus struct{ value string }

var (
	JoinRequestStatusPending  = JoinRequestStatus{value: "PENDING"}
	JoinRequestStatusApproved = JoinRequestStatus{value: "APPROVED"}
	JoinRequestStatusRejected = JoinRequestStatus{value: "REJECTED"}
)

func NewJoinRequestStatus(s string) (JoinRequestStatus, error) {
	switch s {
	case "PENDING", "APPROVED", "REJECTED":
		return JoinRequestStatus{value: s}, nil
	}
	return JoinRequestStatus{}, apperror.NewValidation([]apperror.FieldError{
		{Field: "status", Message: "invalid join request status: " + s},
	})
}

func RestoreJoinRequestStatus(s string) JoinRequestStatus { return JoinRequestStatus{value: s} }
func (s JoinRequestStatus) String() string                { return s.value }
