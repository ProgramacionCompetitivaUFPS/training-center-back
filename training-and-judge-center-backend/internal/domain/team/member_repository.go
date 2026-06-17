package team

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/shared"
)

type MemberRepository interface {
	Save(ctx context.Context, m *TeamMember) error
	FindByTeam(ctx context.Context, teamID string) ([]*TeamMember, error)
	FindByUser(ctx context.Context, userID shared.UserID) ([]*TeamMember, error)
	BulkCountByTeams(ctx context.Context, teamIDs []string) (map[string]int, error)
}
