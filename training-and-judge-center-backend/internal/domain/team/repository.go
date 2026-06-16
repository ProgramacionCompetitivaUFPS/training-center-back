package team

import "context"

type Repository interface {
	Save(ctx context.Context, t *Team) error
	FindByID(ctx context.Context, id string) (*Team, error)
	ExistsByName(ctx context.Context, name TeamName) (bool, error)
}

type MemberRepository interface {
	Save(ctx context.Context, m *TeamMember) error
	FindByTeam(ctx context.Context, teamID string) ([]*TeamMember, error)
}
