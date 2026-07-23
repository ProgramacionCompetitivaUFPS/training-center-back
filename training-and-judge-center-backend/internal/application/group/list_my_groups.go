package group

import (
	"context"
	"log/slog"
	"strings"
	"time"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ListMyGroupsInput struct {
	CurrentUser appshared.CurrentUser
	Role        *string
	Search      string
	SortBy      string
	Order       string
	Page        int
	Limit       int
}

type MyGroupItem struct {
	Group       GroupDTO
	MemberCount int
	MyRole      string
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
	if err := appshared.ValidatePagination(in.Page, in.Limit, maxPageLimit); err != nil {
		return nil, err
	}

	hide, err := uc.preferences.HideGlobalGroup(ctx, in.CurrentUser.ID)
	if err != nil {
		return nil, err
	}

	membership := &domainGroup.MembershipFilter{}
	if in.Role != nil {
		r, err := domainGroup.NewMemberRole(*in.Role)
		if err != nil {
			return nil, apperror.NewValidation([]apperror.FieldError{
				{Field: "role", Message: "invalid role; must be MEMBER or LEAD"},
			})
		}
		membership.RoleFilter = &r
	}

	filters := domainGroup.ListFilters{
		Search:         strings.TrimSpace(in.Search),
		Page:           in.Page,
		Limit:          in.Limit,
		ViewerID:       shared.RestoreUserID(in.CurrentUser.ID),
		ViewerIsAdmin:  false,
		OnlyMyGroups:   membership,
		ExcludeDefault: hide,
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
		if !s.IsMember {
			slog.WarnContext(ctx, "BulkStats returned no membership for listed group; possible TOCTOU",
				"group_id", g.ID(), "viewer_id", filters.ViewerID.Value())
			continue
		}
		items = append(items, MyGroupItem{
			Group:       groupToDTO(g),
			MemberCount: s.Count,
			MyRole:      s.Role.String(),
			JoinedAt:    s.JoinedAt,
		})
	}

	return &ListMyGroupsOutput{
		Groups:     items,
		TotalCount: len(items),
		TotalPages: appshared.CalcTotalPages(total, in.Limit),
		Page:       in.Page,
		Limit:      in.Limit,
	}, nil
}
