package group

import (
	"context"
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
)

type MemberFilters struct {
	Page  int
	Limit int
	Role  *MemberRole
}

type MemberStats struct {
	Count    int
	IsMember bool
	Role     MemberRole
	JoinedAt time.Time
}

type MemberRepository interface {
	Save(ctx context.Context, m *GroupMember) error
	SaveAll(ctx context.Context, members []*GroupMember) error
	FindByGroupAndUser(ctx context.Context, groupID string, userID shared.UserID) (*GroupMember, error)
	FindByGroup(ctx context.Context, groupID string, filters MemberFilters) ([]*GroupMember, int, error)
	Delete(ctx context.Context, groupID string, userID shared.UserID) error
	CountLeads(ctx context.Context, groupID string) (int, error)
	CountMembers(ctx context.Context, groupID string) (int, error)
	ListLeads(ctx context.Context, groupID string) ([]*GroupMember, error)
	BulkStats(ctx context.Context, groupIDs []string, viewerID shared.UserID) (map[string]MemberStats, error)
}
