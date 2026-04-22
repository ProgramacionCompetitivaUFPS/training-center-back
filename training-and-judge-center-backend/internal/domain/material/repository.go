package material

import "context"

type SortField string

const (
	SortByPublishedAt SortField = "publishedAt"
	SortByTitle       SortField = "title"
	SortByRelevance   SortField = "relevance"
)

// ListFilters defines the filtering, sorting and pagination options for listing materials.
// Limit must be between 1 and 100; enforcement is the responsibility of the use case layer.
type ListFilters struct {
	GroupID       string
	ViewerID      *string
	IsAdmin       bool
	IsGroupMember bool
	AuthorID      *string
	Tags          []string
	Pinned        *bool
	SearchQuery   *string
	SortBy        SortField
	Page          int
	Limit         int
}

type Repository interface {
	Save(ctx context.Context, m *Material) error
	FindByID(ctx context.Context, id string) (*Material, error)
	List(ctx context.Context, filters ListFilters) ([]*Material, int, error)
	Delete(ctx context.Context, id string) error
}
