package user

import (
	"context"
	"time"
)

type EmailChangeRequest struct {
	ID        string
	UserID    string
	NewEmail  Email
	Code      string
	Status    RequestStatus
	ExpiresAt time.Time
	CreatedAt time.Time
	UpdatedAt *time.Time
}

func (r *EmailChangeRequest) IsExpired(now time.Time) bool {
	return now.After(r.ExpiresAt) || r.Status == StatusExpired
}

func (r *EmailChangeRequest) MarkAsUsed(now time.Time) {
	r.Status = StatusUsed
	r.UpdatedAt = &now
}

type EmailChangeRepository interface {
	Save(ctx context.Context, req *EmailChangeRequest) error
	FindByID(ctx context.Context, id string) (*EmailChangeRequest, error)
	FindByCodeAndUserID(ctx context.Context, code string, userID string) (*EmailChangeRequest, error)
	InvalidatePendingByUserID(ctx context.Context, userID string) error
	Update(ctx context.Context, req *EmailChangeRequest) error
}
