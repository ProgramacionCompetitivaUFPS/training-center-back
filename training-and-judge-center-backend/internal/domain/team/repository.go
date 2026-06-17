package team

import "context"

type Repository interface {
	Save(ctx context.Context, t *Team) error
	FindByID(ctx context.Context, id string) (*Team, error)
	FindByIDs(ctx context.Context, ids []string) ([]*Team, error)
	ExistsByName(ctx context.Context, name TeamName) (bool, error)
}
