package user

import (
	"context"
	"fmt"
	"time"
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
	return now.After(r.expiresAt) || r.status == StatusExpired
}

func (r *PasswordRecoveryRequest) MarkAsUsed(now time.Time) error {
	if r.status != StatusPending {
		return fmt.Errorf("cannot mark a %s request as used", r.status)
	}
	r.status = StatusUsed
	r.updatedAt = &now
	return nil
}

type PasswordRecoveryRepository interface {
	Save(ctx context.Context, req *PasswordRecoveryRequest) error
	FindByID(ctx context.Context, id string) (*PasswordRecoveryRequest, error)
	Update(ctx context.Context, req *PasswordRecoveryRequest) error
	FindPendingByUserID(ctx context.Context, userID string) (*PasswordRecoveryRequest, error)
	InvalidatePendingByUserID(ctx context.Context, userID string, now time.Time) error
}
