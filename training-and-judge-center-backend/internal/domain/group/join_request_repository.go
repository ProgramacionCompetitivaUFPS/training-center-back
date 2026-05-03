package group

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/shared"
)

type JoinRequestFilters struct {
	Status *JoinRequestStatus
	Page   int
	Limit  int
}

type JoinRequestRepository interface {
	Save(ctx context.Context, r *JoinRequest) error
	FindByID(ctx context.Context, id string) (*JoinRequest, error)
	FindByGroupAndUser(ctx context.Context, groupID string, userID shared.UserID) (*JoinRequest, error)
	FindByGroup(ctx context.Context, groupID string, filters JoinRequestFilters) ([]*JoinRequest, int, error)
	Delete(ctx context.Context, id string) error
}
