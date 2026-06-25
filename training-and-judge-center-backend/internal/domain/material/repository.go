package material

import (
	"context"
	"time"
)

type SortField string

const (
	SortByPublishedAt SortField = "publishedAt"
	SortByTitle       SortField = "title"
	SortByRelevance   SortField = "relevance"
)

// ListFilters defines the filtering, sorting and pagination options for listing materials.
// GroupID is mandatory and must be provided as a separate parameter to List.
// Auth rules (isAdmin, isGroupMember, etc.) must be collapsed into these data-only
// filters by the use case layer before calling the repository.
// Limit must be between 1 and 100; enforcement is the responsibility of the use case layer.
type ListFilters struct {
	// Statuses filters by status; empty means no status constraint (all statuses returned).
	Statuses      []Status
	ViewerID      *string
	AuthorID      *string
	Tags          []string
	Pinned        *bool
	SearchQuery   *string
	PublishedFrom *time.Time
	PublishedTo   *time.Time
	SortBy        SortField
	Page          int
	Limit         int
}

type Repository interface {
	Save(ctx context.Context, m *Material) error
	FindByID(ctx context.Context, id string) (*Material, error)
	List(ctx context.Context, groupID string, filters ListFilters) ([]*Material, int, error)
	Delete(ctx context.Context, id string) error
}
