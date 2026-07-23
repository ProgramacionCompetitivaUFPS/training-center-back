package contest

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
)

type GroupSummary struct {
	ID   string
	Name string
}

type MyContestListItem struct {
	ContestListItem
	Group GroupSummary
}

type ListMyContestsOutput struct {
	Items      []MyContestListItem
	Pagination PaginationOutput
}

type ListMyContestsInput struct {
	CurrentUser appshared.CurrentUser
	Status      *string
	SortBy      string
	Order       string
	Page        int
	Limit       int
}

type ListMyContestsUseCase struct {
	repo                domainContest.Repository
	groupProvider       GroupProvider
	participantProvider ContestParticipantProvider
}

func NewListMyContestsUseCase(
	repo domainContest.Repository,
	groupProvider GroupProvider,
	participantProvider ContestParticipantProvider,
) *ListMyContestsUseCase {
	return &ListMyContestsUseCase{
		repo:                repo,
		groupProvider:       groupProvider,
		participantProvider: participantProvider,
	}
}

func (uc *ListMyContestsUseCase) Execute(ctx context.Context, in ListMyContestsInput) (*ListMyContestsOutput, error) {
	if err := appshared.ValidatePagination(in.Page, in.Limit, maxPageLimit); err != nil {
		return nil, err
	}

	groupIDs, err := uc.groupProvider.ListAccessibleGroupIDs(ctx, in.CurrentUser.ID, in.CurrentUser.IsAdmin())
	if err != nil {
		return nil, err
	}

	filters := buildContestFilters(in.Status, in.SortBy, in.Order, in.Page, in.Limit)

	contests, total, err := uc.repo.ListByGroupIDs(ctx, groupIDs, filters)
	if err != nil {
		return nil, err
	}

	contestIDs := make([]string, len(contests))
	distinctGroupIDs := make([]string, 0, len(contests))
	seenGroups := make(map[string]bool, len(contests))
	for i, c := range contests {
		contestIDs[i] = c.ID()
		gid := c.GroupID().Value()
		if !seenGroups[gid] {
			seenGroups[gid] = true
			distinctGroupIDs = append(distinctGroupIDs, gid)
		}
	}

	participantCounts, err := uc.participantProvider.CountParticipantsBulk(ctx, contestIDs)
	if err != nil {
		return nil, err
	}
	registeredMap, err := uc.participantProvider.IsRegisteredBulk(ctx, contestIDs, in.CurrentUser.ID)
	if err != nil {
		return nil, err
	}
	groupInfos, err := uc.groupProvider.FindByIDs(ctx, distinctGroupIDs)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	items := make([]MyContestListItem, 0, len(contests))
	for _, c := range contests {
		item := MyContestListItem{
			ContestListItem: toContestListItem(c, now, participantCounts[c.ID()], registeredMap[c.ID()]),
			Group:           GroupSummary{ID: c.GroupID().Value()},
		}
		if info, ok := groupInfos[c.GroupID().Value()]; ok {
			item.Group.Name = info.Name
		}
		items = append(items, item)
	}

	return &ListMyContestsOutput{
		Items: items,
		Pagination: PaginationOutput{
			Page:       in.Page,
			Limit:      in.Limit,
			Total:      total,
			TotalPages: appshared.CalcTotalPages(total, in.Limit),
		},
	}, nil
}
