package user

import (
	"fmt"
	"time"

	"github.com/training-judge-center/backend/pkg/apperror"
)

type EmailChangeRequest struct {
	id        string
	userID    string
	newEmail  Email
	code      string
	status    RequestStatus
	expiresAt time.Time
	createdAt time.Time
	updatedAt *time.Time
}

func NewEmailChangeRequest(id, userID string, newEmail Email, code string, now time.Time) (*EmailChangeRequest, error) {
	if id == "" || userID == "" || code == "" {
		return nil, apperror.NewInternal()
	}
	return &EmailChangeRequest{
		id:        id,
		userID:    userID,
		newEmail:  newEmail,
		code:      code,
		status:    StatusPending,
		expiresAt: now.Add(15 * time.Minute),
		createdAt: now,
	}, nil
}

func RestoreEmailChangeRequest(id, userID string, newEmail Email, code string, status RequestStatus, expiresAt, createdAt time.Time, updatedAt *time.Time) *EmailChangeRequest {
	return &EmailChangeRequest{
		id:        id,
		userID:    userID,
		newEmail:  newEmail,
		code:      code,
		status:    status,
		expiresAt: expiresAt,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

func (r *EmailChangeRequest) ID() string                { return r.id }
func (r *EmailChangeRequest) UserID() string            { return r.userID }
func (r *EmailChangeRequest) NewEmail() Email           { return r.newEmail }
func (r *EmailChangeRequest) Code() string              { return r.code }
func (r *EmailChangeRequest) Status() RequestStatus     { return r.status }
func (r *EmailChangeRequest) ExpiresAt() time.Time      { return r.expiresAt }
func (r *EmailChangeRequest) CreatedAt() time.Time      { return r.createdAt }
func (r *EmailChangeRequest) UpdatedAt() *time.Time     { return r.updatedAt }

func (r *EmailChangeRequest) IsExpired(now time.Time) bool {
	return now.After(r.expiresAt) || r.status == StatusExpired
}

func (r *EmailChangeRequest) MarkAsUsed(now time.Time) error {
	if r.status != StatusPending {
		return fmt.Errorf("cannot mark a %s request as used", r.status)
	}
	r.status = StatusUsed
	r.updatedAt = &now
	return nil
}

