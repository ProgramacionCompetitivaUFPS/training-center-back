package group

import "github.com/training-judge-center/backend/pkg/apperror"

const (
	joinRequestStatusPending  = "PENDING"
	joinRequestStatusApproved = "APPROVED"
	joinRequestStatusRejected = "REJECTED"
)

type JoinRequestStatus struct{ value string }

var (
	JoinRequestStatusPending  = JoinRequestStatus{value: joinRequestStatusPending}
	JoinRequestStatusApproved = JoinRequestStatus{value: joinRequestStatusApproved}
	JoinRequestStatusRejected = JoinRequestStatus{value: joinRequestStatusRejected}
)

func NewJoinRequestStatus(s string) (JoinRequestStatus, error) {
	switch s {
	case joinRequestStatusPending, joinRequestStatusApproved, joinRequestStatusRejected:
		return JoinRequestStatus{value: s}, nil
	}
	return JoinRequestStatus{}, apperror.NewValidation([]apperror.FieldError{
		{Field: "status", Message: "invalid join request status: " + s},
	})
}

func RestoreJoinRequestStatus(s string) JoinRequestStatus { return JoinRequestStatus{value: s} }
func (s JoinRequestStatus) String() string                { return s.value }
