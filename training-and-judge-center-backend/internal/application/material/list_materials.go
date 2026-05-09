package material

import (
	"context"
	"fmt"
	"log/slog"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	DefaultPage  = 1
	DefaultLimit = 20
	MaxLimit     = 100
)

type ListMaterialsInput struct {
	CurrentUser appshared.CurrentUser
	GroupID     string
	Pinned      *bool
	Tags        []string
	Page        int
	Limit       int
}

type PaginationData struct {
	TotalCount   int
	CurrentPage  int
	TotalPages   int
	ItemsPerPage int
}

type ListMaterialsOutput struct {
	Materials  []MaterialData
	Pagination PaginationData
}

type ListMaterialsUseCase struct {
	repo            domainMaterial.Repository
	groupVisibility GroupVisibilityProvider
	memberProvider  GroupMemberProvider
	authorProvider  AuthorProvider
}

func NewListMaterialsUseCase(
	repo domainMaterial.Repository,
	groupVisibility GroupVisibilityProvider,
	memberProvider GroupMemberProvider,
	authorProvider AuthorProvider,
) *ListMaterialsUseCase {
	return &ListMaterialsUseCase{
		repo:            repo,
		groupVisibility: groupVisibility,
		memberProvider:  memberProvider,
		authorProvider:  authorProvider,
	}
}

func (uc *ListMaterialsUseCase) Execute(ctx context.Context, in ListMaterialsInput) (*ListMaterialsOutput, error) {
	if in.Page < 1 {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "page", Message: "page must be a positive integer"},
		})
	}
	if in.Limit < 1 || in.Limit > MaxLimit {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "limit", Message: fmt.Sprintf("limit must be between 1 and %d", MaxLimit)},
		})
	}

	visibility, exists, err := uc.groupVisibility.FindVisibility(ctx, in.GroupID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to find group visibility", "error", err, "group_id", in.GroupID)
		return nil, apperror.NewInternal()
	}
	if !exists {
		return nil, apperror.NewNotFound(ErrCodeGroupNotFound, "group not found")
	}

	if err := checkGroupAccess(ctx, uc.memberProvider, in.CurrentUser, in.GroupID, visibility); err != nil {
		return nil, err
	}

	filters, err := uc.buildFilters(ctx, in)
	if err != nil {
		return nil, err
	}

	materials, total, err := uc.repo.List(ctx, in.GroupID, filters)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list materials", "error", err, "group_id", in.GroupID)
		return nil, apperror.NewInternal()
	}

	authorIDs := uniqueAuthorIDs(materials)
	displays, err := uc.authorProvider.GetDisplays(ctx, authorIDs)
	if err != nil {
		slog.ErrorContext(ctx, "failed to resolve author displays", "error", err, "group_id", in.GroupID)
		return nil, apperror.NewInternal()
	}

	items := make([]MaterialData, 0, len(materials))
	for _, m := range materials {
		d := toMaterialData(m)
		if disp := displays[m.AuthorID().Value()]; disp != nil {
			d.Author = &AuthorDTO{Nickname: disp.Nickname, Name: disp.Name}
		}
		items = append(items, d)
	}

	totalPages := total / in.Limit
	if total%in.Limit != 0 {
		totalPages++
	}

	return &ListMaterialsOutput{
		Materials: items,
		Pagination: PaginationData{
			TotalCount:   total,
			CurrentPage:  in.Page,
			TotalPages:   totalPages,
			ItemsPerPage: in.Limit,
		},
	}, nil
}

func (uc *ListMaterialsUseCase) buildFilters(ctx context.Context, in ListMaterialsInput) (domainMaterial.ListFilters, error) {
	filters := domainMaterial.ListFilters{
		Tags:   in.Tags,
		Pinned: in.Pinned,
		SortBy: domainMaterial.SortByPublishedAt,
		Page:   in.Page,
		Limit:  in.Limit,
	}

	if in.CurrentUser.IsAdmin() {
		return filters, nil
	}

	isLead, err := uc.memberProvider.IsLeadOfGroup(ctx, in.CurrentUser.ID, in.GroupID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check lead role", "error", err, "user_id", in.CurrentUser.ID, "group_id", in.GroupID)
		return domainMaterial.ListFilters{}, apperror.NewInternal()
	}

	// Unlike problems (where ViewerModifierID controls draft visibility via authorship),
	// materials tie draft visibility to group role (Lead), not authorship.
	if isLead {
		return filters, nil
	}

	// Members and non-members only see PUBLISHED.
	filters.Statuses = []domainMaterial.Status{domainMaterial.NewStatusPublished()}
	return filters, nil
}

func uniqueAuthorIDs(materials []*domainMaterial.Material) []string {
	seen := make(map[string]struct{}, len(materials))
	ids := make([]string, 0, len(materials))
	for _, m := range materials {
		id := m.AuthorID().Value()
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	return ids
}
