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

const maxLimit = 100


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
	if err := appshared.ValidatePagination(in.Page, in.Limit, maxLimit); err != nil {
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
		var err error
		isMember, err = uc.memberProvider.IsMemberOfGroup(ctx, in.CurrentUser.ID, in.GroupID)
		if err != nil {
			return nil, err
		}
		if !isMember {
			isLead, err := uc.memberProvider.IsLeadOfGroup(ctx, in.CurrentUser.ID, in.GroupID)
			if err != nil {
				return nil, err
			}
			isMember = isLead
		}
	}

	if !group.IsVisible && !isMember && !isAdmin {
		return nil, apperror.NewNotFound(ErrCodeGroupNotFound, "group not found")
	}

	filters := domainContest.ListFilters{
		GroupID: shared.RestoreGroupID(in.GroupID),
		Page:    page,
		Limit:   limit,
	}

	if in.Status != nil {
		switch *in.Status {
		case "SCHEDULED":
			s := domainContest.StatusScheduled
			filters.Status = &s
		case "ACTIVE":
			s := domainContest.StatusActive
			filters.Status = &s
		case "FINISHED":
			s := domainContest.StatusFinished
			filters.Status = &s
		}
	}

	switch in.SortBy {
	case "name":
		filters.SortBy = domainContest.SortByName
	case "createdAt":
		filters.SortBy = domainContest.SortByCreatedAt
	default:
		filters.SortBy = domainContest.SortByStartTime
	}

	if in.Order == "asc" {
		filters.Order = domainContest.OrderAsc
	} else {
		filters.Order = domainContest.OrderDesc
	}

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
		items = append(items, ContestListItem{
			ID:                c.ID(),
			Name:              c.Name().Value(),
			Description:       truncateDescription(c.Description()),
			StartTime:         c.StartTime(),
			EndTime:           c.EndTime(),
			Duration:          c.Duration() * 60,
			Status:            c.Status(now).String(),
			Penalty:           c.Penalty().Value(),
			FreezeMinutes:     c.FreezeMinutes(),
			EnablePostContest: c.EnablePostContest(),
			ParticipantCount:  participantCounts[c.ID()],
			IsRegistered:      registeredMap[c.ID()],
			ProblemCount:      len(c.Problems()),
		})
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
