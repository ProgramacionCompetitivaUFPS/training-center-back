package problem

import "context"

type ErrSlugAlreadyExists struct {
	Slug string
}

func (e *ErrSlugAlreadyExists) Error() string {
	return "slug already exists: " + e.Slug
}

type Repository interface {
	Save(ctx context.Context, p *Problem) error
	FindBySlug(ctx context.Context, slug Slug) (*Problem, error)
	ExistsBySlug(ctx context.Context, slug Slug) (bool, error)
}
