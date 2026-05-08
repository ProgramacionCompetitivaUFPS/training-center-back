package user

import (
	"time"

	"github.com/training-judge-center/backend/pkg/apperror"
)

type PasswordRecoveryRequest struct {
	id        string
	userID    string
	code      string
	status    RequestStatus
	expiresAt time.Time
	createdAt time.Time
	updatedAt *time.Time
}

func NewPasswordRecoveryRequest(id, userID, code string, now time.Time) (*PasswordRecoveryRequest, error) {
	if id == "" || userID == "" || code == "" {
		return nil, apperror.NewInternal()
	}
	t := now.UTC()
	return &PasswordRecoveryRequest{
		id:        id,
		userID:    userID,
		code:      code,
		status:    RequestStatusPending,
		expiresAt: t.Add(RequestExpiryDuration),
		createdAt: t,
	}, nil
}

func RestorePasswordRecoveryRequest(id, userID, code string, status RequestStatus, expiresAt, createdAt time.Time, updatedAt *time.Time) *PasswordRecoveryRequest {
	return &PasswordRecoveryRequest{
		id:        id,
		userID:    userID,
		code:      code,
		status:    status,
		expiresAt: expiresAt,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

func (r *PasswordRecoveryRequest) ID() string            { return r.id }
func (r *PasswordRecoveryRequest) UserID() string        { return r.userID }
func (r *PasswordRecoveryRequest) Code() string          { return r.code }
func (r *PasswordRecoveryRequest) Status() RequestStatus { return r.status }
func (r *PasswordRecoveryRequest) ExpiresAt() time.Time  { return r.expiresAt }
func (r *PasswordRecoveryRequest) CreatedAt() time.Time  { return r.createdAt }
func (r *PasswordRecoveryRequest) UpdatedAt() *time.Time { return r.updatedAt }

func (r *PasswordRecoveryRequest) IsExpired(now time.Time) bool {
	return now.After(r.expiresAt) || r.status == RequestStatusExpired
}

func (r *PasswordRecoveryRequest) MarkAsUsed(now time.Time) error {
	if r.status != RequestStatusPending {
		return apperror.NewConflict(ErrCodePasswordRecoveryNotPending, "password recovery request is not pending")
	}
	r.status = RequestStatusUsed
	t := now.UTC()
	r.updatedAt = &t
	return nil
}

