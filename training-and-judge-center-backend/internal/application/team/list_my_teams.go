package team

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

const maxTeamsPageLimit = 100

type ListMyTeamsInput struct {
	CurrentUser appshared.CurrentUser
	Page        int
	Limit       int
}

type MyTeamItem struct {
	ID          string
	Name        string
	MemberCount int
	JoinedAt    time.Time
	CreatedAt   time.Time
}

type ListMyTeamsOutput struct {
	Teams      []MyTeamItem
	TotalCount int
	TotalPages int
	Page       int
	Limit      int
}

type ListMyTeamsUseCase struct {
	teamRepo   domainTeam.Repository
	memberRepo domainTeam.MemberRepository
}

func NewListMyTeamsUseCase(teamRepo domainTeam.Repository, memberRepo domainTeam.MemberRepository) *ListMyTeamsUseCase {
	return &ListMyTeamsUseCase{teamRepo: teamRepo, memberRepo: memberRepo}
}

func (uc *ListMyTeamsUseCase) Execute(ctx context.Context, in ListMyTeamsInput) (*ListMyTeamsOutput, error) {
	if err := appshared.ValidatePagination(in.Page, in.Limit, maxTeamsPageLimit); err != nil {
		return nil, err
	}

	userID := shared.RestoreUserID(in.CurrentUser.ID)

	memberships, err := uc.memberRepo.FindByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	total := len(memberships)

	start, end := pageSliceBounds(in.Page, in.Limit, total)
	page := memberships[start:end]

	if len(page) == 0 {
		return &ListMyTeamsOutput{
			Teams:      []MyTeamItem{},
			TotalCount: total,
			TotalPages: appshared.CalcTotalPages(total, in.Limit),
			Page:       in.Page,
			Limit:      in.Limit,
		}, nil
	}

	teamIDs := make([]string, len(page))
	for i, m := range page {
		teamIDs[i] = m.TeamID()
	}

	teams, err := uc.teamRepo.FindByIDs(ctx, teamIDs)
	if err != nil {
		return nil, err
	}

	counts, err := uc.memberRepo.BulkCountByTeams(ctx, teamIDs)
	if err != nil {
		return nil, err
	}

	teamByID := make(map[string]*domainTeam.Team, len(teams))
	for _, t := range teams {
		teamByID[t.ID()] = t
	}

	items := make([]MyTeamItem, 0, len(page))
	for _, m := range page {
		t, ok := teamByID[m.TeamID()]
		if !ok {
			continue
		}
		items = append(items, MyTeamItem{
			ID:          t.ID(),
			Name:        t.Name().Value(),
			MemberCount: counts[t.ID()],
			JoinedAt:    m.JoinedAt(),
			CreatedAt:   t.CreatedAt(),
		})
	}

	return &ListMyTeamsOutput{
		Teams:      items,
		TotalCount: total,
		TotalPages: appshared.CalcTotalPages(total, in.Limit),
		Page:       in.Page,
		Limit:      in.Limit,
	}, nil
}

func pageSliceBounds(page, limit, total int) (start, end int) {
	start = (page - 1) * limit
	if start > total {
		start = total
	}
	end = start + limit
	if end > total {
		end = total
	}
	return
}
