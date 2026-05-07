package user

import (
	"context"
	"time"
)

type EmailChangeRepository interface {
	Save(ctx context.Context, req *EmailChangeRequest) error
	FindByID(ctx context.Context, id string) (*EmailChangeRequest, error)
	FindByCodeAndUserID(ctx context.Context, code string, userID string) (*EmailChangeRequest, error)
	InvalidatePendingByUserID(ctx context.Context, userID string, now time.Time) error
	Update(ctx context.Context, req *EmailChangeRequest) error
}
