package team

import "context"

type MemberRepository interface {
	Save(ctx context.Context, m *TeamMember) error
	FindByTeam(ctx context.Context, teamID string) ([]*TeamMember, error)
}
