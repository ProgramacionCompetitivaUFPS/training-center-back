package team

import (
	"context"
	"time"

	"github.com/google/uuid"
	appShared "github.com/training-judge-center/backend/internal/application/shared"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type InviteToTeamInput struct {
	CurrentUser appShared.CurrentUser
	TeamID      string
	Nickname    string
}

type InviteToTeamOutput struct {
	ID          string
	TeamID      string
	InviteeUser UserDisplay
	InvitedBy   UserDisplay
	CreatedAt   time.Time
}

type InviteToTeamUseCase struct {
	teamRepo       domainTeam.Repository
	memberRepo     domainTeam.MemberRepository
	invitationRepo domainTeam.InvitationRepository
	userProvider   UserProvider
}

func NewInviteToTeamUseCase(
	teamRepo domainTeam.Repository,
	memberRepo domainTeam.MemberRepository,
	invitationRepo domainTeam.InvitationRepository,
	userProvider UserProvider,
) *InviteToTeamUseCase {
	return &InviteToTeamUseCase{
		teamRepo:       teamRepo,
		memberRepo:     memberRepo,
		invitationRepo: invitationRepo,
		userProvider:   userProvider,
	}
}

func (uc *InviteToTeamUseCase) Execute(ctx context.Context, in InviteToTeamInput, now time.Time) (*InviteToTeamOutput, error) {
	requesterID := domainShared.RestoreUserID(in.CurrentUser.ID)

	if _, err := uc.teamRepo.FindByID(ctx, in.TeamID); err != nil {
		return nil, err
	}

	if _, err := uc.memberRepo.FindByTeamAndUser(ctx, in.TeamID, requesterID); err != nil {
		if ae, ok := err.(*apperror.AppError); ok && ae.Kind == apperror.KindNotFound {
			return nil, apperror.NewForbidden(domainTeam.ErrCodeNotTeamMember, "You are not a member of this team")
		}
		return nil, err
	}

	invitee, err := uc.userProvider.FindByNickname(ctx, in.Nickname)
	if err != nil {
		return nil, err
	}

	inviteeID := domainShared.RestoreUserID(invitee.ID)

	if _, err := uc.memberRepo.FindByTeamAndUser(ctx, in.TeamID, inviteeID); err == nil {
		return nil, apperror.NewConflict(domainTeam.ErrCodeMemberAlreadyOnTeam, "User is already a member of this team")
	} else if ae, ok := err.(*apperror.AppError); !ok || ae.Kind != apperror.KindNotFound {
		return nil, err
	}

	if _, err := uc.invitationRepo.FindByTeamAndInvitee(ctx, in.TeamID, inviteeID); err == nil {
		return nil, apperror.NewConflict(domainTeam.ErrCodeAlreadyInvited, "User already has a pending invitation to this team")
	} else if ae, ok := err.(*apperror.AppError); !ok || ae.Kind != apperror.KindNotFound {
		return nil, err
	}

	inv, err := domainTeam.NewTeamInvitation(uuid.New().String(), in.TeamID, inviteeID, requesterID, now)
	if err != nil {
		return nil, err
	}

	if err := uc.invitationRepo.Save(ctx, inv); err != nil {
		return nil, err
	}

	displays, err := uc.userProvider.GetDisplays(ctx, []string{invitee.ID, in.CurrentUser.ID})
	if err != nil {
		return nil, err
	}

	nickname := func(id string) string {
		if d := displays[id]; d != nil {
			return d.Nickname
		}
		return ""
	}

	return &InviteToTeamOutput{
		ID:     inv.ID(),
		TeamID: inv.TeamID(),
		InviteeUser: UserDisplay{
			ID:       invitee.ID,
			Nickname: nickname(invitee.ID),
		},
		InvitedBy: UserDisplay{
			ID:       in.CurrentUser.ID,
			Nickname: nickname(in.CurrentUser.ID),
		},
		CreatedAt: inv.CreatedAt(),
	}, nil
}
