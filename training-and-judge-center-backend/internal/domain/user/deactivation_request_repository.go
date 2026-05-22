package user

import (
	"context"
	"time"
)

type DeactivationRequestRepository interface {
	Save(ctx context.Context, req *DeactivationRequest) error
	FindPendingByUserID(ctx context.Context, userID string) (*DeactivationRequest, error)
	FindByID(ctx context.Context, id string) (*DeactivationRequest, error)
	Update(ctx context.Context, req *DeactivationRequest) error
	InvalidatePendingByUserID(ctx context.Context, userID string, now time.Time) error
}
