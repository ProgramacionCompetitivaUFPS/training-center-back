package user

import (
	"context"
	"time"
)

type RequestStatus string

const (
	StatusPending RequestStatus = "PENDING"
	StatusUsed    RequestStatus = "USED"
	StatusExpired RequestStatus = "EXPIRED"
)

type PasswordRecoveryRequest struct {
	ID        string
	UserID    string
	Code      string
	Status    RequestStatus
	ExpiresAt time.Time
	CreatedAt time.Time
	UpdatedAt *time.Time
}

func (r *PasswordRecoveryRequest) IsExpired(now time.Time) bool {
	return now.After(r.ExpiresAt) || r.Status == StatusExpired
}

func (r *PasswordRecoveryRequest) MarkAsUsed(now time.Time) {
	r.Status = StatusUsed
	r.UpdatedAt = &now
}

type PasswordRecoveryRepository interface {
	Save(ctx context.Context, req *PasswordRecoveryRequest) error
	FindByID(ctx context.Context, id string) (*PasswordRecoveryRequest, error)
	Update(ctx context.Context, req *PasswordRecoveryRequest) error
}
