package group

import "github.com/training-judge-center/backend/pkg/apperror"

type JoinRequestStatus string

const (
	JoinRequestStatusPending  JoinRequestStatus = "PENDING"
	JoinRequestStatusApproved JoinRequestStatus = "APPROVED"
	JoinRequestStatusRejected JoinRequestStatus = "REJECTED"
)

func NewJoinRequestStatus(s string) (JoinRequestStatus, error) {
	switch JoinRequestStatus(s) {
	case JoinRequestStatusPending, JoinRequestStatusApproved, JoinRequestStatusRejected:
		return JoinRequestStatus(s), nil
	}
	return "", apperror.NewValidation([]apperror.FieldError{
		{Field: "status", Message: "invalid join request status: " + s},
	})
}

func RestoreJoinRequestStatus(s string) JoinRequestStatus { return JoinRequestStatus(s) }
func (s JoinRequestStatus) String() string                { return string(s) }
