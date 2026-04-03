package problem

import "context"

type ErrSlugAlreadyExists struct {
	Slug string
}

func (e *ErrSlugAlreadyExists) Error() string {
	return "slug already exists: " + e.Slug
}

type ListFilters struct {
	Statuses         []string
	ViewerModifierID *string
	AuthorID         *string
	Tags             []string
	Accessibility    *string
	Page             int
	Limit            int
}

type Repository interface {
	Save(ctx context.Context, p *Problem) error
	FindBySlug(ctx context.Context, slug Slug) (*Problem, error)
	ExistsBySlug(ctx context.Context, slug Slug) (bool, error)
	List(ctx context.Context, filters ListFilters) ([]*Problem, int, error)
}
