package team

import (
	"context"
	"time"

	"github.com/google/uuid"
	appShared "github.com/training-judge-center/backend/internal/application/shared"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RegisterTeamToContestInput struct {
	CurrentUser     appShared.CurrentUser
	ContestID       string
	TeamID          string
	SelectedMembers []string
}

type SelectedMemberDisplay struct {
	ID       string
	Nickname string
}

type RegisterTeamToContestOutput struct {
	ID              string
	ContestID       string
	TeamID          string
	TeamName        string
	SelectedMembers []SelectedMemberDisplay
	RegisteredAt    time.Time
}

type RegisterTeamToContestUseCase struct {
	memberRepo      domainTeam.MemberRepository
	teamRepo        domainTeam.Repository
	contestProvider ContestProvider
	teamParticipRepo domainContest.TeamParticipationRepository
	indivChecker    IndividualRegistrationChecker
	groupChecker    GroupMemberChecker
	userProvider    UserProvider
	txManager       appShared.TransactionManager
}

func NewRegisterTeamToContestUseCase(
	memberRepo domainTeam.MemberRepository,
	teamRepo domainTeam.Repository,
	contestProvider ContestProvider,
	teamParticipRepo domainContest.TeamParticipationRepository,
	indivChecker IndividualRegistrationChecker,
	groupChecker GroupMemberChecker,
	userProvider UserProvider,
	txManager appShared.TransactionManager,
) *RegisterTeamToContestUseCase {
	return &RegisterTeamToContestUseCase{
		memberRepo:       memberRepo,
		teamRepo:         teamRepo,
		contestProvider:  contestProvider,
		teamParticipRepo: teamParticipRepo,
		indivChecker:     indivChecker,
		groupChecker:     groupChecker,
		userProvider:     userProvider,
		txManager:        txManager,
	}
}

func (uc *RegisterTeamToContestUseCase) Execute(ctx context.Context, in RegisterTeamToContestInput) (*RegisterTeamToContestOutput, error) {
	requesterID := domainShared.RestoreUserID(in.CurrentUser.ID)

	// 1. Requester must be a team member.
	if _, err := uc.memberRepo.FindByTeamAndUser(ctx, in.TeamID, requesterID); err != nil {
		if ae, ok := err.(*apperror.AppError); ok && ae.Kind == apperror.KindNotFound {
			return nil, apperror.NewForbidden(domainTeam.ErrCodeNotTeamMember, "you are not a member of this team")
		}
		return nil, err
	}

	// 2. Contest must exist and allow teams.
	contest, err := uc.contestProvider.GetContestInfo(ctx, in.ContestID)
	if err != nil {
		return nil, err
	}
	if !contest.AllowsTeams() {
		return nil, apperror.NewBadRequest(ErrCodeTeamsNotAllowed, "this contest does not allow team registrations")
	}

	// 3. Contest must be SCHEDULED.
	if contest.Status != "SCHEDULED" {
		return nil, apperror.NewConflict(ErrCodeContestNotOpenForRegistration, "contest is not open for registration")
	}

	// Reject duplicate selected members.
	seenMembers := make(map[string]struct{}, len(in.SelectedMembers))
	for _, uid := range in.SelectedMembers {
		if _, dup := seenMembers[uid]; dup {
			return nil, apperror.NewBadRequest(ErrCodeInvalidSelectedMember, "selected members list contains duplicate user IDs")
		}
		seenMembers[uid] = struct{}{}
	}

	// 4. selectedMembers must all be members of the team.
	teamMembers, err := uc.memberRepo.FindByTeam(ctx, in.TeamID)
	if err != nil {
		return nil, err
	}
	memberSet := make(map[string]struct{}, len(teamMembers))
	for _, m := range teamMembers {
		memberSet[m.UserID().Value()] = struct{}{}
	}
	for _, uid := range in.SelectedMembers {
		if _, ok := memberSet[uid]; !ok {
			return nil, apperror.NewBadRequest(ErrCodeInvalidSelectedMember, "one or more selected members are not members of this team")
		}
	}

	// 5. Count within [min, max].
	if len(in.SelectedMembers) < contest.TeamSizeMin || len(in.SelectedMembers) > contest.TeamSizeMax {
		return nil, apperror.NewBadRequest(ErrCodeInvalidTeamSize,
			"selected member count must be between the contest's team size limits")
	}

	// 6. All selectedMembers must be group members (contest always belongs to a group).
	groupMembership, err := uc.groupChecker.AreUsersGroupMembers(ctx, contest.GroupID, in.SelectedMembers)
	if err != nil {
		return nil, err
	}
	for _, uid := range in.SelectedMembers {
		if !groupMembership[uid] {
			return nil, apperror.NewForbidden(ErrCodeMemberNotInGroup, "one or more selected members are not members of the contest group")
		}
	}

	// 7. Atomic: check conflicts then save.
	newID := uuid.New().String()
	now := time.Now()

	participation, err := domainContest.NewContestTeamParticipation(newID, in.ContestID, in.TeamID, in.SelectedMembers, now)
	if err != nil {
		return nil, err
	}

	if err := uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		// 7a. No individual registration for any selectedMember.
		indivMap, err := uc.indivChecker.AreUsersRegisteredIndividually(txCtx, in.ContestID, in.SelectedMembers)
		if err != nil {
			return err
		}
		for _, uid := range in.SelectedMembers {
			if indivMap[uid] {
				return apperror.NewConflict(ErrCodeMemberAlreadyRegistered, "one or more selected members are already individually registered to this contest")
			}
		}

		// 7b. No selectedMember in another team for this contest.
		inAnotherTeam, err := uc.teamParticipRepo.AreUsersSelectedInAnyTeam(txCtx, in.ContestID, in.SelectedMembers, "")
		if err != nil {
			return err
		}
		for _, uid := range in.SelectedMembers {
			if inAnotherTeam[uid] {
				return apperror.NewConflict(ErrCodeMemberInAnotherTeam, "one or more selected members are already registered in another team for this contest")
			}
		}

		return uc.teamParticipRepo.Save(txCtx, participation)
	}); err != nil {
		return nil, err
	}

	// Build output.
	team, err := uc.teamRepo.FindByID(ctx, in.TeamID)
	if err != nil {
		return nil, err
	}

	displays, err := uc.userProvider.GetDisplays(ctx, in.SelectedMembers)
	if err != nil {
		return nil, err
	}
	members := make([]SelectedMemberDisplay, 0, len(in.SelectedMembers))
	for _, uid := range in.SelectedMembers {
		d := displays[uid]
		nickname := ""
		if d != nil {
			nickname = d.Nickname
		}
		members = append(members, SelectedMemberDisplay{ID: uid, Nickname: nickname})
	}

	return &RegisterTeamToContestOutput{
		ID:              participation.ID(),
		ContestID:       participation.ContestID(),
		TeamID:          participation.TeamID(),
		TeamName:        team.Name().Value(),
		SelectedMembers: members,
		RegisteredAt:    participation.RegisteredAt(),
	}, nil
}
