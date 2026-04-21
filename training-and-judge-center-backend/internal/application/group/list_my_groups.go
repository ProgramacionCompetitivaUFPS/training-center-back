package group

import (
	"context"
	"fmt"
	"math"
	"strings"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ListMyGroupsInput struct {
	CurrentUser shared.CurrentUser
	Role        *string
	Search      string
	SortBy      string
	Order       string
	Page        int
	Limit       int
}

// MyGroupItem: un grupo donde el viewer es miembro.
type MyGroupItem struct {
	Group       *domainGroup.Group
	MemberCount int
	MyRole      domainGroup.MemberRole
	JoinedAt    string // RFC3339
}

type ListMyGroupsOutput struct {
	Groups     []MyGroupItem
	TotalCount int
	TotalPages int
	Page       int
	Limit      int
}

type ListMyGroupsUseCase struct {
	repo        domainGroup.Repository
	memberRepo  domainGroup.MemberRepository
	preferences PreferencesReader
}

func NewListMyGroupsUseCase(repo domainGroup.Repository, memberRepo domainGroup.MemberRepository, prefs PreferencesReader) *ListMyGroupsUseCase {
	return &ListMyGroupsUseCase{repo: repo, memberRepo: memberRepo, preferences: prefs}
}

func (uc *ListMyGroupsUseCase) Execute(ctx context.Context, in ListMyGroupsInput) (*ListMyGroupsOutput, error) {
	if in.Page < 1 {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "page", Message: "page must be a positive integer"},
		})
	}
	if in.Limit < 1 || in.Limit > MaxPageLimit {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "limit", Message: fmt.Sprintf("limit must be between 1 and %d", MaxPageLimit)},
		})
	}

	hide, err := uc.preferences.HideGlobalGroup(ctx, in.CurrentUser.ID)
	if err != nil {
		return nil, err
	}

	filters := domainGroup.ListFilters{
		Search:         strings.TrimSpace(in.Search),
		Page:           in.Page,
		Limit:          in.Limit,
		ViewerID:       shared.RestoreUserID(in.CurrentUser.ID),
		ViewerIsAdmin:  false, // /me/groups IGNORA permisos implícitos de admin (FR-VG-025)
		OnlyMyGroups:   true,
		ExcludeDefault: hide,
	}

	if in.Role != nil {
		r, err := domainGroup.NewMemberRole(*in.Role)
		if err != nil {
			return nil, apperror.NewValidation([]apperror.FieldError{
				{Field: "role", Message: "invalid role; must be MEMBER or LEAD"},
			})
		}
		filters.RoleFilter = &r
	}

	sortBy, err := parseSort(in.SortBy, domainGroup.SortByName, []domainGroup.SortField{
		domainGroup.SortByName, domainGroup.SortByJoinedAt, domainGroup.SortByMemberCount,
	})
	if err != nil {
		return nil, err
	}
	filters.SortBy = sortBy

	order, err := parseOrder(in.Order)
	if err != nil {
		return nil, err
	}
	filters.Order = order

	groups, total, err := uc.repo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	items := make([]MyGroupItem, 0, len(groups))
	for _, g := range groups {
		m, err := uc.memberRepo.FindByGroupAndUser(ctx, g.ID(), filters.ViewerID)
		if err != nil {
			return nil, err
		}
		if m == nil {
			continue // defensivo: el repo ya filtró por OnlyMyGroups
		}
		count, err := uc.memberRepo.CountMembers(ctx, g.ID())
		if err != nil {
			return nil, err
		}
		items = append(items, MyGroupItem{
			Group:       g,
			MemberCount: count,
			MyRole:      m.Role(),
			JoinedAt:    m.JoinedAt().Format("2006-01-02T15:04:05Z"),
		})
	}

	totalPages := 0
	if in.Limit > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(in.Limit)))
	}

	return &ListMyGroupsOutput{
		Groups:     items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       in.Page,
		Limit:      in.Limit,
	}, nil
}
