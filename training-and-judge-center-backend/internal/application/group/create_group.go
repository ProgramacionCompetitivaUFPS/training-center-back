package group

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type CreateGroupInput struct {
	Name        string
	Description *string
	JoinMode    string
	Visibility  string
	CurrentUser appshared.CurrentUser
}

type CreateGroupOutput struct {
	ID          string
	Name        string
	Description *string
	JoinPolicy  string
	Visibility  string
	IsDefault   bool
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreateGroupUseCase struct {
	repo       domainGroup.Repository
	memberRepo domainGroup.MemberRepository
	txManager  appshared.TransactionManager
}

func NewCreateGroupUseCase(repo domainGroup.Repository, memberRepo domainGroup.MemberRepository, txManager appshared.TransactionManager) *CreateGroupUseCase {
	return &CreateGroupUseCase{repo: repo, memberRepo: memberRepo, txManager: txManager}
}

func (uc *CreateGroupUseCase) Execute(ctx context.Context, input CreateGroupInput) (*CreateGroupOutput, error) {
	if input.CurrentUser.Role != shared.RoleCoach && !input.CurrentUser.IsAdmin() {
		return nil, apperror.NewForbidden(
			ErrCodeInsufficientPermissions,
			"Only Admin and Coach users can create groups",
		)
	}

	var fieldErrs []apperror.FieldError

	groupName, err := domainGroup.NewGroupName(input.Name)
	if err := apperror.AccumulateFieldErrors(err, &fieldErrs); err != nil {
		return nil, err
	}

	joinPolicy, err := domainGroup.NewJoinPolicy(input.JoinMode)
	if err := apperror.AccumulateFieldErrors(err, &fieldErrs); err != nil {
		return nil, err
	}

	visibility, err := domainGroup.NewVisibility(input.Visibility)
	if err := apperror.AccumulateFieldErrors(err, &fieldErrs); err != nil {
		return nil, err
	}

	if len(fieldErrs) > 0 {
		return nil, apperror.NewValidation(fieldErrs)
	}

	exists, err := uc.repo.ExistsByName(ctx, groupName)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperror.NewConflict(domainGroup.ErrCodeNameAlreadyExists, "A group with this name already exists")
	}

	now := time.Now()
	newID := uuid.New().String()
	creatorID := shared.RestoreUserID(input.CurrentUser.ID)
	g, err := domainGroup.NewGroup(newID, groupName, input.Description, visibility, joinPolicy, creatorID, now)
	if err != nil {
		return nil, err
	}

	if err := uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if err := uc.repo.Save(txCtx, g); err != nil {
			slog.ErrorContext(txCtx, "failed to save new group", "error", err, "group_id", newID)
			return apperror.NewInternal()
		}
		newLeadMemberID := uuid.New().String()
		lead, err := domainGroup.NewGroupMember(newLeadMemberID, newID, creatorID, domainGroup.MemberRoleLead, now)
		if err != nil {
			return err
		}
		return uc.memberRepo.Save(txCtx, lead)
	}); err != nil {
		return nil, err
	}

	return &CreateGroupOutput{
		ID:          g.ID(),
		Name:        g.Name().Value(),
		Description: g.Description(),
		JoinPolicy:  g.JoinPolicy().String(),
		Visibility:  g.Visibility().String(),
		IsDefault:   g.IsDefault(),
		CreatedBy:   g.CreatedBy().Value(),
		CreatedAt:   g.CreatedAt(),
		UpdatedAt:   g.UpdatedAt(),
	}, nil
}
