package team

import (
	"context"
	"time"

	"github.com/google/uuid"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type CreateTeamInput struct {
	Name        string
	CurrentUser appshared.CurrentUser
}

type MemberOutput struct {
	UserID   string
	Nickname string
	JoinedAt time.Time
}

type CreateTeamOutput struct {
	ID        string
	Name      string
	CreatedBy string
	CreatedAt time.Time
	Members   []MemberOutput
}

type CreateTeamUseCase struct {
	teamRepo     domainTeam.Repository
	memberRepo   domainTeam.MemberRepository
	userProvider UserProvider
	txManager    appshared.TransactionManager
}

func NewCreateTeamUseCase(
	teamRepo domainTeam.Repository,
	memberRepo domainTeam.MemberRepository,
	userProvider UserProvider,
	txManager appshared.TransactionManager,
) *CreateTeamUseCase {
	return &CreateTeamUseCase{
		teamRepo:     teamRepo,
		memberRepo:   memberRepo,
		userProvider: userProvider,
		txManager:    txManager,
	}
}

func (uc *CreateTeamUseCase) Execute(ctx context.Context, in CreateTeamInput) (*CreateTeamOutput, error) {
	name, err := domainTeam.NewTeamName(in.Name)
	if err != nil {
		return nil, err
	}

	exists, err := uc.teamRepo.ExistsByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperror.NewConflict(domainTeam.ErrCodeTeamNameExists, "A team with this name already exists")
	}

	now := time.Now()
	creatorID := shared.RestoreUserID(in.CurrentUser.ID)

	team, err := domainTeam.NewTeam(uuid.New().String(), name, creatorID, now)
	if err != nil {
		return nil, err
	}
	member, err := domainTeam.NewTeamMember(uuid.New().String(), team.ID(), creatorID, now)
	if err != nil {
		return nil, err
	}

	if err := uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if err := uc.teamRepo.Save(txCtx, team); err != nil {
			return err
		}
		return uc.memberRepo.Save(txCtx, member)
	}); err != nil {
		return nil, err
	}

	display, err := uc.userProvider.GetDisplay(ctx, creatorID.Value())
	if err != nil {
		return nil, err
	}

	nickname := ""
	if display != nil {
		nickname = display.Nickname
	}

	return &CreateTeamOutput{
		ID:        team.ID(),
		Name:      team.Name().Value(),
		CreatedBy: team.CreatedBy().Value(),
		CreatedAt: team.CreatedAt(),
		Members: []MemberOutput{
			{
				UserID:   creatorID.Value(),
				Nickname: nickname,
				JoinedAt: member.JoinedAt(),
			},
		},
	}, nil
}
