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

// ListedGroup es el item enriquecido para la respuesta de lista.
// Además del agregado Group incluye memberCount y userRole.
type ListedGroup struct {
	Group       *domainGroup.Group
	MemberCount int
	UserRole    *domainGroup.MemberRole // nil si no es miembro
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

	sortBy, err := parseSort(in.SortBy, domainGroup.SortByName, validListSortFields())
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

	items := make([]ListedGroup, 0, len(groups))
	for _, g := range groups {
		count, err := uc.memberRepo.CountMembers(ctx, g.ID())
		if err != nil {
			return nil, err
		}

		var role *domainGroup.MemberRole
		m, err := uc.memberRepo.FindByGroupAndUser(ctx, g.ID(), filters.ViewerID)
		if err == nil && m != nil {
			r := m.Role()
			role = &r
		}
		items = append(items, ListedGroup{
			Group:       g,
			MemberCount: count,
			UserRole:    role,
		})
	}

	totalPages := 0
	if in.Limit > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(in.Limit)))
	}

	return &ListGroupsOutput{
		Groups:     items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       in.Page,
		Limit:      in.Limit,
	}, nil
}

// Helpers compartidos entre use cases.

func validListSortFields() []domainGroup.SortField {
	return []domainGroup.SortField{
		domainGroup.SortByName,
		domainGroup.SortByCreatedAt,
		domainGroup.SortByMemberCount,
	}
}

func parseSort(raw string, def domainGroup.SortField, allowed []domainGroup.SortField) (domainGroup.SortField, error) {
	if raw == "" {
		return def, nil
	}
	for _, f := range allowed {
		if string(f) == raw {
			return f, nil
		}
	}
	names := make([]string, len(allowed))
	for i, f := range allowed {
		names[i] = string(f)
	}
	return "", apperror.NewValidation([]apperror.FieldError{
		{Field: "sortBy", Message: fmt.Sprintf("invalid sortBy; must be one of: %s", strings.Join(names, ", "))},
	})
}

func parseOrder(raw string) (domainGroup.SortOrder, error) {
	switch raw {
	case "":
		return domainGroup.OrderAsc, nil
	case "asc":
		return domainGroup.OrderAsc, nil
	case "desc":
		return domainGroup.OrderDesc, nil
	default:
		return "", apperror.NewValidation([]apperror.FieldError{
			{Field: "order", Message: "invalid order; must be asc or desc"},
		})
	}
}
