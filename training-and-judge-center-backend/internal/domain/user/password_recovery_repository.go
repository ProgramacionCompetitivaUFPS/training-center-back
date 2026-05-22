package user

import (
	"context"
	"time"
)

type PasswordRecoveryRepository interface {
	Save(ctx context.Context, req *PasswordRecoveryRequest) error
	FindByID(ctx context.Context, id string) (*PasswordRecoveryRequest, error)
	Update(ctx context.Context, req *PasswordRecoveryRequest) error
	FindPendingByUserID(ctx context.Context, userID string) (*PasswordRecoveryRequest, error)
	InvalidatePendingByUserID(ctx context.Context, userID string, now time.Time) error
}
