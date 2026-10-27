package team

import (
	"context"

	appShared "github.com/training-judge-center/backend/internal/application/shared"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GetTeamRegistrationEligibilityInput struct {
	CurrentUser appShared.CurrentUser
	GroupID     string
	ContestID   string
	TeamID      string
}

// MemberEligibility tells whether one roster member can be selected when the
// team registers to the contest. Reason is empty when Eligible is true;
// otherwise it is the error code RegisterTeamToContest would reject them with.
type MemberEligibility struct {
	ID       string
	Nickname string
	Eligible bool
	Reason   string
}

type GetTeamRegistrationEligibilityOutput struct {
	// TeamAlreadyRegistered is true when this team already has a registration
	// for the contest; registering it again fails with TEAM_ALREADY_REGISTERED.
	TeamAlreadyRegistered bool
	Members               []MemberEligibility
}

// GetTeamRegistrationEligibilityUseCase runs, for every member of the team's
// roster, the per-member checks RegisterTeamToContest enforces, so the client
// can offer only the members that can actually be selected. Registration
// re-checks everything, since eligibility may change in between.
type GetTeamRegistrationEligibilityUseCase struct {
	memberRepo       domainTeam.MemberRepository
	contestProvider  ContestProvider
	teamParticipRepo domainContest.TeamParticipationRepository
	indivChecker     IndividualRegistrationChecker
	groupChecker     GroupMemberChecker
	userProvider     UserProvider
}

func NewGetTeamRegistrationEligibilityUseCase(
	memberRepo domainTeam.MemberRepository,
	contestProvider ContestProvider,
	teamParticipRepo domainContest.TeamParticipationRepository,
	indivChecker IndividualRegistrationChecker,
	groupChecker GroupMemberChecker,
	userProvider UserProvider,
) *GetTeamRegistrationEligibilityUseCase {
	return &GetTeamRegistrationEligibilityUseCase{
		memberRepo:       memberRepo,
		contestProvider:  contestProvider,
		teamParticipRepo: teamParticipRepo,
		indivChecker:     indivChecker,
		groupChecker:     groupChecker,
		userProvider:     userProvider,
	}
}

func (uc *GetTeamRegistrationEligibilityUseCase) Execute(ctx context.Context, in GetTeamRegistrationEligibilityInput) (*GetTeamRegistrationEligibilityOutput, error) {
	requesterID := domainShared.RestoreUserID(in.CurrentUser.ID)

	// Requester must be a team member.
	if _, err := uc.memberRepo.FindByTeamAndUser(ctx, in.TeamID, requesterID); err != nil {
		if ae, ok := err.(*apperror.AppError); ok && ae.Kind == apperror.KindNotFound {
			return nil, apperror.NewForbidden(domainTeam.ErrCodeNotTeamMember, "you are not a member of this team")
		}
		return nil, err
	}

	// Contest must belong to the given group.
	contest, err := uc.contestProvider.GetContestInfo(ctx, in.ContestID)
	if err != nil {
		return nil, err
	}
	if contest.GroupID != in.GroupID {
		return nil, apperror.NewNotFound(ErrCodeContestNotFound, "contest not found")
	}

	alreadyRegistered := true
	if _, err := uc.teamParticipRepo.FindByContestAndTeam(ctx, in.ContestID, in.TeamID); err != nil {
		if ae, ok := err.(*apperror.AppError); !ok || ae.Kind != apperror.KindNotFound {
			return nil, err
		}
		alreadyRegistered = false
	}

	teamMembers, err := uc.memberRepo.FindByTeam(ctx, in.TeamID)
	if err != nil {
		return nil, err
	}
	userIDs := make([]string, len(teamMembers))
	for i, m := range teamMembers {
		userIDs[i] = m.UserID().Value()
	}

	inGroup, err := uc.groupChecker.AreUsersGroupMembers(ctx, contest.GroupID, userIDs)
	if err != nil {
		return nil, err
	}
	registeredIndividually, err := uc.indivChecker.AreUsersRegisteredIndividually(ctx, in.ContestID, userIDs)
	if err != nil {
		return nil, err
	}
	// Excluding this team keeps its own registration from flagging its members.
	inAnotherTeam, err := uc.teamParticipRepo.AreUsersSelectedInAnyTeam(ctx, in.ContestID, userIDs, in.TeamID)
	if err != nil {
		return nil, err
	}
	displays, err := uc.userProvider.GetDisplays(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	members := make([]MemberEligibility, 0, len(userIDs))
	for _, uid := range userIDs {
		reason := ""
		switch {
		case !inGroup[uid]:
			reason = ErrCodeMemberNotInGroup
		case registeredIndividually[uid]:
			reason = ErrCodeMemberAlreadyRegistered
		case inAnotherTeam[uid]:
			reason = ErrCodeMemberInAnotherTeam
		}
		nickname := ""
		if d := displays[uid]; d != nil {
			nickname = d.Nickname
		}
		members = append(members, MemberEligibility{
			ID:       uid,
			Nickname: nickname,
			Eligible: reason == "",
			Reason:   reason,
		})
	}

	return &GetTeamRegistrationEligibilityOutput{TeamAlreadyRegistered: alreadyRegistered, Members: members}, nil
}
