package user

import (
	"context"
	"time"
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

func (r *EmailChangeRequest) MarkAsUsed(now time.Time) {
	r.status = StatusUsed
	r.updatedAt = &now
}

type EmailChangeRepository interface {
	Save(ctx context.Context, req *EmailChangeRequest) error
	FindByID(ctx context.Context, id string) (*EmailChangeRequest, error)
	FindByCodeAndUserID(ctx context.Context, code string, userID string) (*EmailChangeRequest, error)
	InvalidatePendingByUserID(ctx context.Context, userID string) error
	Update(ctx context.Context, req *EmailChangeRequest) error
}
