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

const (
	DefaultPage  = 1
	DefaultLimit = 20
	MaxLimit     = 100
)

type ContestListItem struct {
	ID                string
	Name              string
	Description       *string
	StartTime         time.Time
	EndTime           time.Time
	Duration          int // in seconds (spec FR-010)
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
	page := in.Page
	if page < 1 {
		page = DefaultPage
	}
	limit := in.Limit
	if limit < 1 {
		limit = DefaultLimit
	} else if limit > MaxLimit {
		limit = MaxLimit
	}

	group, err := uc.groupProvider.FindByID(ctx, in.GroupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, apperror.NewNotFound(ErrCodeGroupNotFound, "group not found")
	}

	isAdmin := in.CurrentUser.IsAdmin()

	isMember, err := uc.memberProvider.IsMemberOfGroup(ctx, in.CurrentUser.ID, in.GroupID)
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

	now := time.Now()
	items := make([]ContestListItem, 0, len(contests))
	for _, c := range contests {
		participantCount, err := uc.participantProvider.CountParticipants(ctx, c.ID())
		if err != nil {
			return nil, err
		}
		isRegistered, err := uc.participantProvider.IsRegistered(ctx, c.ID(), in.CurrentUser.ID)
		if err != nil {
			return nil, err
		}
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
			ParticipantCount:  participantCount,
			IsRegistered:      isRegistered,
			ProblemCount:      len(c.Problems()),
		})
	}

	totalPages := total / limit
	if total%limit != 0 {
		totalPages++
	}

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
