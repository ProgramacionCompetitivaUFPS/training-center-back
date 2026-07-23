package contest

import (
	"context"
	"time"
	"unicode/utf8"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const maxPageLimit = 100


type ContestListItem struct {
	ID                string
	Name              string
	Description       *string
	StartTime         time.Time
	EndTime           time.Time
	Duration          int // seconds; domain Duration() returns minutes, multiplied here
	Status            string
	Penalty           int
	FreezeMinutes     int
	EnablePostContest bool
	ParticipantCount  int
	IsRegistered      bool
	ProblemCount      int
}

type PaginationOutput struct {
	Page       int
	Limit      int
	Total      int
	TotalPages int
}

type ListContestsOutput struct {
	Items      []ContestListItem
	Pagination PaginationOutput
}

func truncateDescription(d *string) *string {
	if d == nil {
		return nil
	}
	const max = 200
	if utf8.RuneCountInString(*d) <= max {
		return d
	}
	runes := []rune(*d)
	s := string(runes[:max])
	return &s
}

type ListContestsInput struct {
	CurrentUser appshared.CurrentUser
	GroupID     string
	Status      *string
	SortBy      string
	Order       string
	Page        int
	Limit       int
}

type ListContestsUseCase struct {
	repo                domainContest.Repository
	groupProvider       GroupProvider
	memberProvider      GroupMemberProvider
	participantProvider ContestParticipantProvider
}

func NewListContestsUseCase(
	repo domainContest.Repository,
	groupProvider GroupProvider,
	memberProvider GroupMemberProvider,
	participantProvider ContestParticipantProvider,
) *ListContestsUseCase {
	return &ListContestsUseCase{
		repo:                repo,
		groupProvider:       groupProvider,
		memberProvider:      memberProvider,
		participantProvider: participantProvider,
	}
}

func (uc *ListContestsUseCase) Execute(ctx context.Context, in ListContestsInput) (*ListContestsOutput, error) {
	if err := appshared.ValidatePagination(in.Page, in.Limit, maxPageLimit); err != nil {
		return nil, err
	}
	page := in.Page
	limit := in.Limit

	group, err := uc.groupProvider.FindByID(ctx, in.GroupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, apperror.NewNotFound(ErrCodeGroupNotFound, "group not found")
	}

	isAdmin := in.CurrentUser.IsAdmin()

	isMember := false
	if !isAdmin {
		role, err := uc.memberProvider.GetMemberRole(ctx, in.CurrentUser.ID, in.GroupID)
		if err != nil {
			return nil, err
		}
		isMember = role != nil
	}

	if !group.IsVisible && !isMember && !isAdmin {
		return nil, apperror.NewNotFound(ErrCodeGroupNotFound, "group not found")
	}

	filters := buildContestFilters(in.Status, in.SortBy, in.Order, page, limit)
	filters.GroupID = shared.RestoreGroupID(in.GroupID)

	contests, total, err := uc.repo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	contestIDs := make([]string, len(contests))
	for i, c := range contests {
		contestIDs[i] = c.ID()
	}
	participantCounts, err := uc.participantProvider.CountParticipantsBulk(ctx, contestIDs)
	if err != nil {
		return nil, err
	}
	registeredMap, err := uc.participantProvider.IsRegisteredBulk(ctx, contestIDs, in.CurrentUser.ID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	items := make([]ContestListItem, 0, len(contests))
	for _, c := range contests {
		items = append(items, toContestListItem(c, now, participantCounts[c.ID()], registeredMap[c.ID()]))
	}

	totalPages := appshared.CalcTotalPages(total, limit)

	return &ListContestsOutput{
		Items: items,
		Pagination: PaginationOutput{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}
