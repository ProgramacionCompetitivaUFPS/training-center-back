package group

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/shared"
)

type ListFilters struct {
	Search     string
	Visibility *Visibility
	Page       int
	Limit      int
}

type MemberFilters struct {
	Page  int
	Limit int
	Role  *MemberRole
}

type Repository interface {
	Save(ctx context.Context, g *Group) error
	FindByID(ctx context.Context, id string) (*Group, error)
	ExistsByName(ctx context.Context, name GroupName) (bool, error)
	FindDefault(ctx context.Context) (*Group, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filters ListFilters) ([]*Group, int, error)
}

type MemberRepository interface {
	Save(ctx context.Context, m *GroupMember) error
	SaveAll(ctx context.Context, members []*GroupMember) error
	FindByGroupAndUser(ctx context.Context, groupID string, userID shared.UserID) (*GroupMember, error)
	FindByGroup(ctx context.Context, groupID string, filters MemberFilters) ([]*GroupMember, int, error)
	Delete(ctx context.Context, groupID string, userID shared.UserID) error
	CountLeads(ctx context.Context, groupID string) (int, error)
}
