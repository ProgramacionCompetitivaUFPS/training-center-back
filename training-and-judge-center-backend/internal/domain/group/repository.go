package group

import (
	"context"

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

type Repository interface {
	Save(ctx context.Context, g *Group) error
	Update(ctx context.Context, g *Group) error
	FindByID(ctx context.Context, id string) (*Group, error)
	ExistsByName(ctx context.Context, name GroupName) (bool, error)
	FindDefault(ctx context.Context) (*Group, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filters ListFilters) ([]*Group, int, error)
}

