package group

import (
	"context"
	"log/slog"
	"strings"
	"time"

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

type MyGroupItem struct {
	Group       *domainGroup.Group
	MemberCount int
	MyRole      domainGroup.MemberRole
	JoinedAt    time.Time
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
	if err := validatePagination(in.Page, in.Limit); err != nil {
		return nil, err
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
		ViewerIsAdmin:  false,
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

	groupIDs := make([]string, len(groups))
	for i, g := range groups {
		groupIDs[i] = g.ID()
	}
	stats, err := uc.memberRepo.BulkStats(ctx, groupIDs, filters.ViewerID)
	if err != nil {
		return nil, err
	}

	items := make([]MyGroupItem, 0, len(groups))
	for _, g := range groups {
		s := stats[g.ID()]
		if s.Membership == nil {
			slog.WarnContext(ctx, "BulkStats returned no membership for listed group; possible TOCTOU",
				"group_id", g.ID(), "viewer_id", filters.ViewerID.Value())
			continue
		}
		items = append(items, MyGroupItem{
			Group:       g,
			MemberCount: s.Count,
			MyRole:      s.Membership.Role(),
			JoinedAt:    s.Membership.JoinedAt(),
		})
	}

	return &ListMyGroupsOutput{
		Groups:     items,
		TotalCount: len(items),
		TotalPages: calcTotalPages(total, in.Limit),
		Page:       in.Page,
		Limit:      in.Limit,
	}, nil
}
