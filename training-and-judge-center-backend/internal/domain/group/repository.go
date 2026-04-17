package group

import "context"

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
	FindByGroupAndUser(ctx context.Context, groupID, userID string) (*GroupMember, error)
	FindByGroup(ctx context.Context, groupID string, filters MemberFilters) ([]*GroupMember, int, error)
	Delete(ctx context.Context, groupID, userID string) error
	CountLeads(ctx context.Context, groupID string) (int, error)
}
