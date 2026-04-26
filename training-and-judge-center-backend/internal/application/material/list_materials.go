package material

import (
	"context"
	"log/slog"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	DefaultPage  = 1
	DefaultLimit = 20
	MaxLimit     = 100
)

type ListMaterialsInput struct {
	CurrentUser shared.CurrentUser
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

type ListMaterials struct {
	repo            domainMaterial.Repository
	groupVisibility GroupVisibilityProvider
	memberProvider  GroupMemberProvider
	authorProvider  AuthorProvider
}

func NewListMaterials(
	repo domainMaterial.Repository,
	groupVisibility GroupVisibilityProvider,
	memberProvider GroupMemberProvider,
	authorProvider AuthorProvider,
) *ListMaterials {
	return &ListMaterials{
		repo:            repo,
		groupVisibility: groupVisibility,
		memberProvider:  memberProvider,
		authorProvider:  authorProvider,
	}
}

func (uc *ListMaterials) Execute(ctx context.Context, in ListMaterialsInput) (*ListMaterialsOutput, error) {
	if in.Page < 1 {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "page", Message: "page must be a positive integer"},
		})
	}
	if in.Limit < 1 || in.Limit > MaxLimit {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "limit", Message: "limit must be between 1 and 100"},
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

	// Reuse the same group access check from GetMaterial logic.
	if visibility != "VISIBLE" && !in.CurrentUser.IsAdmin() {
		isMember, err := uc.memberProvider.IsMemberOfGroup(ctx, in.CurrentUser.ID, in.GroupID)
		if err != nil {
			slog.ErrorContext(ctx, "failed to check group membership", "error", err)
			return nil, apperror.NewInternal()
		}
		if !isMember {
			return nil, apperror.NewForbidden(ErrCodeInsufficientPerms, "you do not have permission to view materials in this group")
		}
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
		slog.WarnContext(ctx, "failed to resolve author displays", "error", err)
		displays = map[string]*AuthorDisplay{}
	}

	items := make([]MaterialData, 0, len(materials))
	for _, m := range materials {
		d := toMaterialData(m)
		if ad := displays[m.AuthorID().Value()]; ad != nil {
			d.Author = &AuthorData{Nickname: ad.Nickname, Name: ad.Name}
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

func (uc *ListMaterials) buildFilters(ctx context.Context, in ListMaterialsInput) (domainMaterial.ListFilters, error) {
	filters := domainMaterial.ListFilters{
		Tags:   in.Tags,
		Pinned: in.Pinned,
		SortBy: domainMaterial.SortByPublishedAt,
		Page:   in.Page,
		Limit:  in.Limit,
	}

	if in.CurrentUser.IsAdmin() {
		filters.Statuses = []domainMaterial.Status{
			domainMaterial.NewStatusDraft(),
			domainMaterial.NewStatusPublished(),
		}
		return filters, nil
	}

	isLead, err := uc.memberProvider.IsLeadOfGroup(ctx, in.CurrentUser.ID, in.GroupID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check lead role", "error", err)
		return filters, apperror.NewInternal()
	}

	if isLead {
		filters.Statuses = []domainMaterial.Status{
			domainMaterial.NewStatusDraft(),
			domainMaterial.NewStatusPublished(),
		}
	} else {
		filters.Statuses = []domainMaterial.Status{domainMaterial.NewStatusPublished()}
	}

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
