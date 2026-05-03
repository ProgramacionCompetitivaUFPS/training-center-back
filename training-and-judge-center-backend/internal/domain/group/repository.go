package group

import (
	"context"
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
)

type SortField string

const (
	SortByName        SortField = "name"
	SortByCreatedAt   SortField = "createdAt"
	SortByMemberCount SortField = "memberCount"
	SortByJoinedAt    SortField = "joinedAt"
)

type SortOrder string

const (
	OrderAsc  SortOrder = "asc"
	OrderDesc SortOrder = "desc"
)

type MembershipFilter struct {
	RoleFilter *MemberRole
}

type ListFilters struct {
	Search     string
	Visibility *Visibility
	JoinPolicy *JoinPolicy
	SortBy     SortField
	Order      SortOrder
	Page       int
	Limit      int

	ViewerID      shared.UserID
	ViewerIsAdmin bool

	OnlyMyGroups *MembershipFilter

	ExcludeDefault bool
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

type MemberStats struct {
	Count    int
	IsMember bool
	Role     MemberRole
	JoinedAt time.Time
}

type JoinRequestFilters struct {
	Status *JoinRequestStatus
	Page   int
	Limit  int
}

type JoinRequestRepository interface {
	Save(ctx context.Context, r *JoinRequest) error
	FindByID(ctx context.Context, id string) (*JoinRequest, error)
	FindByGroupAndUser(ctx context.Context, groupID string, userID shared.UserID) (*JoinRequest, error)
	FindByGroup(ctx context.Context, groupID string, filters JoinRequestFilters) ([]*JoinRequest, int, error)
	Delete(ctx context.Context, id string) error
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
