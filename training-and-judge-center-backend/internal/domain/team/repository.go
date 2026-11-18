package team

import "context"

type Repository interface {
	Save(ctx context.Context, t *Team) error
	FindByID(ctx context.Context, id string) (*Team, error)
	FindByIDs(ctx context.Context, ids []string) ([]*Team, error)
	ExistsByName(ctx context.Context, name TeamName) (bool, error)
	// UpdateName saves the team's current name. It fails with TEAM_NAME_EXISTS when another team has it.
	UpdateName(ctx context.Context, t *Team) error
	// LockByID takes a row lock on the team until the end of the transaction in ctx, so operations that
	// would otherwise race (two members removing each other) run one after the other. It fails with
	// TEAM_NOT_FOUND when the team does not exist.
	LockByID(ctx context.Context, id string) error
}
