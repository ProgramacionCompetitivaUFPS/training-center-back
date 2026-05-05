package contest

import "context"

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
	GroupID string
	Status  *Status
	Search  string
	SortBy  SortField
	Order   SortOrder
	Page    int
	Limit   int
}

type Repository interface {
	Save(ctx context.Context, c *Contest) error
	FindByID(ctx context.Context, id string) (*Contest, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filters ListFilters) ([]*Contest, int, error)
}
