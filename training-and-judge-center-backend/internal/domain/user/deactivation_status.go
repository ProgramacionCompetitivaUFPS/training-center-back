package user

import "github.com/training-judge-center/backend/pkg/apperror"

const (
	deactivationStatusPending   = "PENDING"
	deactivationStatusConfirmed = "CONFIRMED"
	deactivationStatusExpired   = "EXPIRED"
	deactivationStatusBlocked   = "BLOCKED"
)

type DeactivationStatus struct{ value string }

var (
	DeactivationStatusPending   = DeactivationStatus{value: deactivationStatusPending}
	DeactivationStatusConfirmed = DeactivationStatus{value: deactivationStatusConfirmed}
	DeactivationStatusExpired   = DeactivationStatus{value: deactivationStatusExpired}
	DeactivationStatusBlocked   = DeactivationStatus{value: deactivationStatusBlocked}
)

func NewDeactivationStatus(raw string) (DeactivationStatus, error) {
	switch raw {
	case deactivationStatusPending, deactivationStatusConfirmed,
		deactivationStatusExpired, deactivationStatusBlocked:
		return DeactivationStatus{value: raw}, nil
	default:
		return DeactivationStatus{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "status", Message: "invalid deactivation status: " + raw},
		})
	}
}

func RestoreDeactivationStatus(raw string) DeactivationStatus {
	return DeactivationStatus{value: raw}
}

func (s DeactivationStatus) String() string { return s.value }
