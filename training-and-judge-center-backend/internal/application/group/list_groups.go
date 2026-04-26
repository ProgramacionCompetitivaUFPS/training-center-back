package group

import (
	"context"
	"strings"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	MaxPageLimit     = 50
	DefaultPageLimit = 20
)

type ListGroupsInput struct {
	CurrentUser shared.CurrentUser
	Search      string
	Visibility  *string
	JoinPolicy  *string
	SortBy      string
	Order       string
	Page        int
	Limit       int
}

type ListedGroup struct {
	Group       *domainGroup.Group
	MemberCount int
	UserRole    *domainGroup.MemberRole
}

type ListGroupsOutput struct {
	Groups     []ListedGroup
	TotalCount int
	TotalPages int
	Page       int
	Limit      int
}

type ListGroupsUseCase struct {
	repo       domainGroup.Repository
	memberRepo domainGroup.MemberRepository
}

func NewListGroupsUseCase(repo domainGroup.Repository, memberRepo domainGroup.MemberRepository) *ListGroupsUseCase {
	return &ListGroupsUseCase{repo: repo, memberRepo: memberRepo}
}

func (uc *ListGroupsUseCase) Execute(ctx context.Context, in ListGroupsInput) (*ListGroupsOutput, error) {
	if err := validatePagination(in.Page, in.Limit); err != nil {
		return nil, err
	}

	filters := domainGroup.ListFilters{
		Search:        strings.TrimSpace(in.Search),
		Page:          in.Page,
		Limit:         in.Limit,
		ViewerID:      shared.RestoreUserID(in.CurrentUser.ID),
		ViewerIsAdmin: in.CurrentUser.IsAdmin(),
	}

	if in.Visibility != nil {
		v, err := domainGroup.NewVisibility(*in.Visibility)
		if err != nil {
			return nil, apperror.NewValidation([]apperror.FieldError{
				{Field: "visibility", Message: "invalid visibility; must be VISIBLE or NOT_VISIBLE"},
			})
		}
		filters.Visibility = &v
	}

	if in.JoinPolicy != nil {
		jp, err := domainGroup.NewJoinPolicy(*in.JoinPolicy)
		if err != nil {
			return nil, apperror.NewValidation([]apperror.FieldError{
				{Field: "joinPolicy", Message: "invalid joinPolicy; must be INVITE, REQUEST or OPEN"},
			})
		}
		filters.JoinPolicy = &jp
	}

	sortBy, err := parseSort(in.SortBy, domainGroup.SortByName, []domainGroup.SortField{
		domainGroup.SortByName,
		domainGroup.SortByCreatedAt,
		domainGroup.SortByMemberCount,
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

	items := make([]ListedGroup, 0, len(groups))
	for _, g := range groups {
		s := stats[g.ID()]
		var role *domainGroup.MemberRole
		if s.IsMember {
			r := s.Role
			role = &r
		}
		items = append(items, ListedGroup{
			Group:       g,
			MemberCount: s.Count,
			UserRole:    role,
		})
	}

	return &ListGroupsOutput{
		Groups:     items,
		TotalCount: total,
		TotalPages: calcTotalPages(total, in.Limit),
		Page:       in.Page,
		Limit:      in.Limit,
	}, nil
}
