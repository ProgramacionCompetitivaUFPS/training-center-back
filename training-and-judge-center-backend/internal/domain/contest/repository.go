package contest

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/shared"
)

type SortField string

const (
	SortByName      SortField = "name"
	SortByStartTime SortField = "startTime"
	SortByCreatedAt SortField = "createdAt"
)

type SortOrder string

const (
	OrderAsc  SortOrder = "asc"
	OrderDesc SortOrder = "desc"
)

type ListFilters struct {
	GroupID shared.GroupID
	Status  *Status
	Search  string
	SortBy  SortField
	Order   SortOrder
	Page    int
	Limit   int
}

type Repository interface {
	// Create inserts a new contest. Returns an error if the id already exists.
	Create(ctx context.Context, c *Contest) error
	// Update persists changes to an existing contest using c.UpdatedAt() as an
	// optimistic lock token. Returns ErrCodeContestConflict if another write
	// landed between the read and this call.
	Update(ctx context.Context, c *Contest) error
	FindByID(ctx context.Context, id string) (*Contest, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filters ListFilters) ([]*Contest, int, error)
	// ListByGroupIDs lists contests scoped to groupIDs, ignoring filters.GroupID.
	// groupIDs == nil means "no group restriction" (all contests, for Admins).
	// groupIDs == []string{} means "no accessible groups" (zero results).
	ListByGroupIDs(ctx context.Context, groupIDs []string, filters ListFilters) ([]*Contest, int, error)
}
