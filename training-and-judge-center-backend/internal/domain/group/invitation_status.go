package group

import "github.com/training-judge-center/backend/pkg/apperror"

const (
	invitationStatusPending  = "PENDING"
	invitationStatusAccepted = "ACCEPTED"
	invitationStatusRevoked  = "REVOKED"
	invitationStatusExpired  = "EXPIRED"
)

type InvitationStatus struct{ value string }

var (
	InvitationStatusPending  = InvitationStatus{value: invitationStatusPending}
	InvitationStatusAccepted = InvitationStatus{value: invitationStatusAccepted}
	InvitationStatusRevoked  = InvitationStatus{value: invitationStatusRevoked}
	InvitationStatusExpired  = InvitationStatus{value: invitationStatusExpired}
)

func NewInvitationStatus(s string) (InvitationStatus, error) {
	switch s {
	case invitationStatusPending, invitationStatusAccepted, invitationStatusRevoked, invitationStatusExpired:
		return InvitationStatus{value: s}, nil
	}
	return InvitationStatus{}, apperror.NewValidation([]apperror.FieldError{
		{Field: "status", Message: "invalid invitation status: " + s},
	})
}

func RestoreInvitationStatus(s string) InvitationStatus { return InvitationStatus{value: s} }
func (s InvitationStatus) String() string               { return s.value }
