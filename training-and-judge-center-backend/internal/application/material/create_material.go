package material

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
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
	Material MaterialData
}

type CreateMaterial struct {
	repo           domainMaterial.Repository
	groupProvider  GroupProvider
	memberProvider GroupMemberProvider
}

func NewCreateMaterial(
	repo domainMaterial.Repository,
	groupProvider GroupProvider,
	memberProvider GroupMemberProvider,
) *CreateMaterial {
	return &CreateMaterial{
		repo:           repo,
		groupProvider:  groupProvider,
		memberProvider: memberProvider,
	}
}

func (uc *CreateMaterial) Execute(ctx context.Context, in CreateMaterialInput) (*CreateMaterialOutput, error) {
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

	title, titleErr := domainMaterial.NewTitle(in.Title)
	if err := apperror.AccumulateFieldErrors(titleErr, &fieldErrs); err != nil {
		return nil, err
	}

	var content domainMaterial.Content
	if in.Content != nil {
		var contentErr error
		content, contentErr = domainMaterial.NewContent(*in.Content)
		if err := apperror.AccumulateFieldErrors(contentErr, &fieldErrs); err != nil {
			return nil, err
		}
	} else {
		content = domainMaterial.NewEmptyContent()
	}

	tags, tagsErr := domainMaterial.NewTags(in.Tags)
	if err := apperror.AccumulateFieldErrors(tagsErr, &fieldErrs); err != nil {
		return nil, err
	}

	if len(fieldErrs) > 0 {
		return nil, apperror.NewValidation(fieldErrs)
	}

	m, err := domainMaterial.NewMaterial(
		uuid.New().String(),
		in.GroupID,
		shared.RestoreUserID(in.CurrentUser.ID),
		title,
		content,
		tags,
		nil,
	)
	if err != nil {
		slog.ErrorContext(ctx, "unexpected error constructing material", "error", err, "group_id", in.GroupID)
		return nil, apperror.NewInternal()
	}

	if err := uc.repo.Save(ctx, m); err != nil {
		slog.ErrorContext(ctx, "failed to save material", "error", err, "material_id", m.ID())
		return nil, apperror.NewInternal()
	}

	return &CreateMaterialOutput{Material: toMaterialData(m)}, nil
}
