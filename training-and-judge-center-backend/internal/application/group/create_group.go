package group

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type CreateGroupInput struct {
	Name        string
	Description *string
	JoinMode    string
	Visibility  string
	CurrentUser shared.CurrentUser
}

type CreateGroupResult struct {
	Group *domainGroup.Group
}

type CreateGroupUseCase struct {
	repo       domainGroup.Repository
	memberRepo domainGroup.MemberRepository
	txManager  TransactionManager
}

func NewCreateGroupUseCase(repo domainGroup.Repository, memberRepo domainGroup.MemberRepository, txManager TransactionManager) *CreateGroupUseCase {
	return &CreateGroupUseCase{repo: repo, memberRepo: memberRepo, txManager: txManager}
}

func (uc *CreateGroupUseCase) Execute(ctx context.Context, input CreateGroupInput) (*CreateGroupResult, error) {
	if input.CurrentUser.Role != shared.RoleCoach && !input.CurrentUser.IsAdmin() {
		return nil, apperror.NewForbidden(
			domainGroup.ErrCodeInsufficientPermissions,
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

	newID := uuid.New().String()
	creatorID := shared.RestoreUserID(input.CurrentUser.ID)
	g, err := domainGroup.NewGroup(newID, groupName, input.Description, visibility, joinPolicy, creatorID, nil)
	if err != nil {
		return nil, err
	}

	if err := uc.txManager.WithTx(ctx, func(ctx context.Context) error {
		if err := uc.repo.Save(ctx, g); err != nil {
			slog.ErrorContext(ctx, "failed to save new group", "error", err, "group_id", newID)
			return apperror.NewInternal()
		}
		lead, err := domainGroup.NewGroupMember(uuid.New().String(), newID, creatorID, domainGroup.MemberRoleLead, nil)
		if err != nil {
			return err
		}
		return uc.memberRepo.Save(ctx, lead)
	}); err != nil {
		return nil, err
	}

	return &CreateGroupResult{Group: g}, nil
}
