package material

import (
	"context"
	"strings"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const maxPageLimit = 100

type ListMaterialsInput struct {
	CurrentUser   appshared.CurrentUser
	GroupID       string
	Query         string
	Author        string
	PublishedFrom *time.Time
	PublishedTo   *time.Time
	Pinned        *bool
	Tags          []string
	Sort          string
	Page          int
	Limit         int
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
	repo             domainMaterial.Repository
	groupVisibility  GroupVisibilityProvider
	memberProvider   GroupMemberProvider
	authorProvider   AuthorProvider
	authorIDProvider AuthorIDProvider
}

func NewListMaterialsUseCase(
	repo domainMaterial.Repository,
	groupVisibility GroupVisibilityProvider,
	memberProvider GroupMemberProvider,
	authorProvider AuthorProvider,
	authorIDProvider AuthorIDProvider,
) *ListMaterialsUseCase {
	return &ListMaterialsUseCase{
		repo:             repo,
		groupVisibility:  groupVisibility,
		memberProvider:   memberProvider,
		authorProvider:   authorProvider,
		authorIDProvider: authorIDProvider,
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

	// Resolve author nickname → userID. Unknown nickname → empty result set.
	var authorID *string
	if in.Author != "" {
		id, found, err := uc.authorIDProvider.FindIDByNickname(ctx, in.Author)
		if err != nil {
			return nil, err
		}
		if !found {
			return &ListMaterialsOutput{
				Materials: []MaterialData{},
				Pagination: PaginationData{
					TotalCount:   0,
					CurrentPage:  in.Page,
					TotalPages:   appshared.CalcTotalPages(0, in.Limit),
					ItemsPerPage: in.Limit,
				},
			}, nil
		}
		authorID = &id
	}

	filters, err := uc.buildFilters(ctx, in, authorID)
	if err != nil {
		return nil, err
	}

	materials, total, err := uc.repo.List(ctx, in.GroupID, filters)
	if err != nil {
		return nil, err
	}

	displays, err := uc.authorProvider.GetDisplays(ctx, uniqueAuthorIDs(materials))

	items := make([]MaterialData, 0, len(materials))
	for _, m := range materials {
		d := toMaterialData(m)
		if err == nil {
			if disp := displays[m.AuthorID().Value()]; disp != nil {
				d.Author = &AuthorDTO{Nickname: disp.Nickname, Name: disp.Name}
			}
		}
		items = append(items, d)
	}

	return &ListMaterialsOutput{
		Materials: items,
		Pagination: PaginationData{
			TotalCount:   total,
			CurrentPage:  in.Page,
			TotalPages:   appshared.CalcTotalPages(total, in.Limit),
			ItemsPerPage: in.Limit,
		},
	}, nil
}

func (uc *ListMaterialsUseCase) buildFilters(ctx context.Context, in ListMaterialsInput, authorID *string) (domainMaterial.ListFilters, error) {
	sortBy, err := resolveSortBy(in.Sort, in.Query)
	if err != nil {
		return domainMaterial.ListFilters{}, err
	}

	if in.PublishedFrom != nil && in.PublishedTo != nil && in.PublishedFrom.After(*in.PublishedTo) {
		return domainMaterial.ListFilters{}, apperror.NewBadRequest(ErrCodeInvalidDateRange, "publishedFrom must be before or equal to publishedTo")
	}

	var searchQuery *string
	if q := strings.TrimSpace(in.Query); q != "" {
		searchQuery = &q
	}

	filters := domainMaterial.ListFilters{
		AuthorID:      authorID,
		Tags:          in.Tags,
		Pinned:        in.Pinned,
		SearchQuery:   searchQuery,
		PublishedFrom: in.PublishedFrom,
		PublishedTo:   in.PublishedTo,
		SortBy:        sortBy,
		Page:          in.Page,
		Limit:         in.Limit,
	}

	if in.CurrentUser.IsAdmin() {
		return filters, nil
	}

	isLead, err := uc.memberProvider.IsLeadOfGroup(ctx, in.CurrentUser.ID, in.GroupID)
	if err != nil {
		return domainMaterial.ListFilters{}, err
	}

	if isLead {
		return filters, nil
	}

	filters.Statuses = []domainMaterial.Status{domainMaterial.NewStatusPublished()}
	return filters, nil
}

// resolveSortBy converts the raw sort string to a domain SortField.
// Default: relevance when q is present, publishedAt otherwise.
func resolveSortBy(sort, query string) (domainMaterial.SortField, error) {
	switch sort {
	case "":
		if strings.TrimSpace(query) != "" {
			return domainMaterial.SortByRelevance, nil
		}
		return domainMaterial.SortByPublishedAt, nil
	case "relevance":
		return domainMaterial.SortByRelevance, nil
	case "publishedAt":
		return domainMaterial.SortByPublishedAt, nil
	case "title":
		return domainMaterial.SortByTitle, nil
	default:
		return "", apperror.NewBadRequest(ErrCodeInvalidSort, "sort must be one of: relevance, publishedAt, title")
	}
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
