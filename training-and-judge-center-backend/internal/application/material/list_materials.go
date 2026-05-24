package material

import (
	"context"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const maxPageLimit = 100


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
	if err := appshared.ValidatePagination(in.Page, in.Limit, maxPageLimit); err != nil {
		return nil, err
	}

	visibility, exists, err := uc.groupVisibility.FindVisibility(ctx, in.GroupID)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	authorIDs := uniqueAuthorIDs(materials)
	displays, err := uc.authorProvider.GetDisplays(ctx, authorIDs)
	if err != nil {
		return nil, err
	}

	items := make([]MaterialData, 0, len(materials))
	for _, m := range materials {
		d := toMaterialData(m)
		if disp := displays[m.AuthorID().Value()]; disp != nil {
			d.Author = &AuthorDTO{Nickname: disp.Nickname, Name: disp.Name}
		}
		items = append(items, d)
	}

	totalPages := appshared.CalcTotalPages(total, in.Limit)

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
		return domainMaterial.ListFilters{}, err
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
