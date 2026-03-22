package user

import (
	"context"
	"time"
)

type DeactivationStatus string

const (
	DeactivationStatusPending   DeactivationStatus = "PENDING"
	DeactivationStatusConfirmed DeactivationStatus = "CONFIRMED"
	DeactivationStatusExpired   DeactivationStatus = "EXPIRED"
	DeactivationStatusBlocked   DeactivationStatus = "BLOCKED"
)

type DeactivationRequest struct {
	ID               string
	UserID           string
	VerificationCode string
	ExpiresAt        time.Time
	Attempts         int
	BlockedUntil     *time.Time
	Status           DeactivationStatus
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type DeactivationAuditLog struct {
	ID               string
	UserID           string
	OriginalEmail    string
	OriginalNickname string
	OccurredAt       time.Time
	IP               *string
	UserAgent        *string
}

type DeactivationRequestRepository interface {
	Save(ctx context.Context, req *DeactivationRequest) error
	FindPendingByUserID(ctx context.Context, userID string) (*DeactivationRequest, error)
	FindByID(ctx context.Context, id string) (*DeactivationRequest, error)
	Update(ctx context.Context, req *DeactivationRequest) error
	InvalidatePendingByUserID(ctx context.Context, userID string, now time.Time) error
}

type DeactivationAuditLogRepository interface {
	Save(ctx context.Context, log *DeactivationAuditLog) error
}
