package team

import (
	"context"
	"time"

	appShared "github.com/training-judge-center/backend/internal/application/shared"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ListTeamRegistrationsInput struct {
	CurrentUser appShared.CurrentUser
	ContestID   string
	GroupID     string
	Page        int
	Limit       int
}

type TeamRegistrationItem struct {
	ID              string
	ContestID       string
	TeamID          string
	TeamName        string
	SelectedMembers []SelectedMemberDisplay
	RegisteredAt    time.Time
}

type ListTeamRegistrationsOutput struct {
	Items []TeamRegistrationItem
	Total int
}

type ListTeamRegistrationsUseCase struct {
	contestProvider  ContestProvider
	groupChecker     GroupMemberChecker
	teamParticipRepo domainContest.TeamParticipationRepository
	teamRepo         domainTeam.Repository
	userProvider     UserProvider
}

func NewListTeamRegistrationsUseCase(
	contestProvider ContestProvider,
	groupChecker GroupMemberChecker,
	teamParticipRepo domainContest.TeamParticipationRepository,
	teamRepo domainTeam.Repository,
	userProvider UserProvider,
) *ListTeamRegistrationsUseCase {
	return &ListTeamRegistrationsUseCase{
		contestProvider:  contestProvider,
		groupChecker:     groupChecker,
		teamParticipRepo: teamParticipRepo,
		teamRepo:         teamRepo,
		userProvider:     userProvider,
	}
}

func (uc *ListTeamRegistrationsUseCase) Execute(ctx context.Context, in ListTeamRegistrationsInput) (*ListTeamRegistrationsOutput, error) {
	requesterID := domainShared.RestoreUserID(in.CurrentUser.ID)

	// Must be a member of the contest's group.
	membership, err := uc.groupChecker.AreUsersGroupMembers(ctx, in.GroupID, []string{requesterID.Value()})
	if err != nil {
		return nil, err
	}
	if !membership[requesterID.Value()] && !in.CurrentUser.IsAdmin() {
		return nil, apperror.NewForbidden(ErrCodeMemberNotInGroup, "you are not a member of this group")
	}

	// Contest must belong to the given group.
	contest, err := uc.contestProvider.GetContestInfo(ctx, in.ContestID)
	if err != nil {
		return nil, err
	}
	if contest.GroupID != in.GroupID {
		return nil, apperror.NewNotFound(ErrCodeContestNotFound, "contest not found")
	}

	participations, total, err := uc.teamParticipRepo.List(ctx, in.ContestID, in.Page, in.Limit)
	if err != nil {
		return nil, err
	}

	// Collect all team IDs and user IDs for bulk lookup.
	teamIDs := make([]string, 0, len(participations))
	allUserIDs := make([]string, 0)
	for _, p := range participations {
		teamIDs = append(teamIDs, p.TeamID())
		allUserIDs = append(allUserIDs, p.SelectedMembers()...)
	}

	teams, err := uc.teamRepo.FindByIDs(ctx, teamIDs)
	if err != nil {
		return nil, err
	}
	teamMap := make(map[string]string, len(teams))
	for _, t := range teams {
		teamMap[t.ID()] = t.Name().Value()
	}

	userDisplays, err := uc.userProvider.GetDisplays(ctx, allUserIDs)
	if err != nil {
		return nil, err
	}

	items := make([]TeamRegistrationItem, 0, len(participations))
	for _, p := range participations {
		members := make([]SelectedMemberDisplay, 0, len(p.SelectedMembers()))
		for _, uid := range p.SelectedMembers() {
			nickname := ""
			if d := userDisplays[uid]; d != nil {
				nickname = d.Nickname
			}
			members = append(members, SelectedMemberDisplay{ID: uid, Nickname: nickname})
		}
		items = append(items, TeamRegistrationItem{
			ID:              p.ID(),
			ContestID:       p.ContestID(),
			TeamID:          p.TeamID(),
			TeamName:        teamMap[p.TeamID()],
			SelectedMembers: members,
			RegisteredAt:    p.RegisteredAt(),
		})
	}

	return &ListTeamRegistrationsOutput{Items: items, Total: total}, nil
}
