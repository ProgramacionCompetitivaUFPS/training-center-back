package material

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type CreateMaterialInput struct {
	CurrentUser shared.CurrentUser
	GroupID     string
	Title       string
	Content     *string
	Tags        []string
}

type CreateMaterialOutput struct {
	Material *material.Material
}

type CreateMaterialUseCase struct {
	repo          material.Repository
	groupProvider GroupProvider
	memberProvider GroupMemberProvider
}

func NewCreateMaterialUseCase(
	repo material.Repository,
	groupProvider GroupProvider,
	memberProvider GroupMemberProvider,
) *CreateMaterialUseCase {
	return &CreateMaterialUseCase{
		repo:           repo,
		groupProvider:  groupProvider,
		memberProvider: memberProvider,
	}
}

func (uc *CreateMaterialUseCase) Execute(ctx context.Context, in CreateMaterialInput) (*CreateMaterialOutput, error) {
	exists, err := uc.groupProvider.Exists(ctx, in.GroupID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check group existence", "error", err, "group_id", in.GroupID)
		return nil, apperror.NewInternal()
	}
	if !exists {
		return nil, apperror.NewNotFound(ErrCodeGroupNotFound, "group not found")
	}

	if !in.CurrentUser.IsAdmin() {
		isLead, err := uc.memberProvider.IsLeadOfGroup(ctx, in.CurrentUser.ID, in.GroupID)
		if err != nil {
			slog.ErrorContext(ctx, "failed to check group membership", "error", err, "user_id", in.CurrentUser.ID, "group_id", in.GroupID)
			return nil, apperror.NewInternal()
		}
		if !isLead {
			return nil, apperror.NewForbidden(ErrCodeInsufficientPerms, "only group leads can create materials")
		}
	}

	var fieldErrs []apperror.FieldError

	title, err := material.NewTitle(in.Title)
	if err := apperror.AccumulateFieldErrors(err, &fieldErrs); err != nil {
		return nil, err
	}

	contentStr := ""
	if in.Content != nil {
		contentStr = *in.Content
	}
	content, err := material.NewContent(contentStr)
	if err := apperror.AccumulateFieldErrors(err, &fieldErrs); err != nil {
		return nil, err
	}

	tags, err := material.NewTags(in.Tags)
	if err := apperror.AccumulateFieldErrors(err, &fieldErrs); err != nil {
		return nil, err
	}

	if len(fieldErrs) > 0 {
		return nil, apperror.NewValidation(fieldErrs)
	}

	m, err := material.NewMaterial(
		uuid.New().String(),
		in.GroupID,
		shared.RestoreUserID(in.CurrentUser.ID),
		title,
		content,
		tags,
		nil,
	)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.Save(ctx, m); err != nil {
		slog.ErrorContext(ctx, "failed to save material", "error", err)
		return nil, apperror.NewInternal()
	}

	return &CreateMaterialOutput{Material: m}, nil
}
