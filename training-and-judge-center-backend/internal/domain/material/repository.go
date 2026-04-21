package material

import "context"

type ListFilters struct {
	GroupID       string
	ViewerID      *string
	IsAdmin       bool
	IsGroupMember bool
	AuthorID      *string
	Tags          []string
	Pinned        *bool
	SearchQuery   *string
	SortBy        string
	Page          int
	Limit         int
}

type Repository interface {
	Save(ctx context.Context, m *Material) error
	FindByID(ctx context.Context, id string) (*Material, error)
	List(ctx context.Context, filters ListFilters) ([]*Material, int, error)
	Delete(ctx context.Context, id string) error
}
