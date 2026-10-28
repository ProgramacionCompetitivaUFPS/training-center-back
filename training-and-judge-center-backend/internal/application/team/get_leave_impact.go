package team

import (
	"context"

	appShared "github.com/training-judge-center/backend/internal/application/shared"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GetLeaveImpactInput struct {
	CurrentUser appShared.CurrentUser
	TeamID      string
}

type GetLeaveImpactOutput struct {
	// RegistrationsToCancel are the team's registrations in contests that have not started that
	// leaving would cancel, because the team would be left without the minimum number of players.
	RegistrationsToCancel []RegistrationAtRisk
}

// GetLeaveImpactUseCase tells a member what leaving the team would cost the team, so the
// confirmation can say it before it happens.
type GetLeaveImpactUseCase struct {
	memberRepo domainTeam.MemberRepository
	cleaner    ScheduledParticipationCleaner
}

func NewGetLeaveImpactUseCase(memberRepo domainTeam.MemberRepository, cleaner ScheduledParticipationCleaner) *GetLeaveImpactUseCase {
	return &GetLeaveImpactUseCase{memberRepo: memberRepo, cleaner: cleaner}
}

func (uc *GetLeaveImpactUseCase) Execute(ctx context.Context, in GetLeaveImpactInput) (*GetLeaveImpactOutput, error) {
	userID := domainShared.RestoreUserID(in.CurrentUser.ID)

	if _, err := uc.memberRepo.FindByTeamAndUser(ctx, in.TeamID, userID); err != nil {
		if ae, ok := err.(*apperror.AppError); ok && ae.Kind == apperror.KindNotFound {
			return nil, apperror.NewNotFound(domainTeam.ErrCodeNotTeamMember, "You are not a member of this team")
		}
		return nil, err
	}

	atRisk, err := uc.cleaner.RegistrationsLosingMinimum(ctx, in.TeamID, in.CurrentUser.ID)
	if err != nil {
		return nil, err
	}
	return &GetLeaveImpactOutput{RegistrationsToCancel: atRisk}, nil
}
