package problem

import "context"

type ListFilters struct {
	Statuses         []Status
	ViewerModifierID *string
	AuthorID         *string
	Tags             []string
	Accessibility    *Accessibility
	Page             int
	Limit            int
}

type Repository interface {
	Save(ctx context.Context, p *Problem) error
	FindBySlug(ctx context.Context, slug Slug) (*Problem, error)
	ExistsBySlug(ctx context.Context, slug Slug) (bool, error)
	List(ctx context.Context, filters ListFilters) ([]*Problem, int, error)
	Delete(ctx context.Context, id string) error
}
