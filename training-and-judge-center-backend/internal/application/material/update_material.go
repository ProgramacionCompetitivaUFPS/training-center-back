package material

import (
	"context"
	"errors"
	"log/slog"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type UpdateMaterialInput struct {
	CurrentUser shared.CurrentUser
	GroupID     string
	MaterialID  string
	Title       *string
	Content     *string
	Tags        *[]string // nil = no change; &[]string{} = clear all tags
}

type UpdateMaterialOutput struct {
	Material MaterialData
}

type UpdateMaterial struct {
	repo          domainMaterial.Repository
	groupProvider GroupProvider
}

func NewUpdateMaterial(
	repo domainMaterial.Repository,
	groupProvider GroupProvider,
) *UpdateMaterial {
	return &UpdateMaterial{repo: repo, groupProvider: groupProvider}
}

func (uc *UpdateMaterial) Execute(ctx context.Context, in UpdateMaterialInput) (*UpdateMaterialOutput, error) {
	exists, err := uc.groupProvider.Exists(ctx, in.GroupID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check group existence", "error", err, "group_id", in.GroupID)
		return nil, apperror.NewInternal()
	}
	if !exists {
		return nil, apperror.NewNotFound(ErrCodeGroupNotFound, "group not found")
	}

	m, err := uc.repo.FindByID(ctx, in.MaterialID)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) && appErr.Code == domainMaterial.ErrCodeMaterialNotFound {
			return nil, err
		}
		slog.ErrorContext(ctx, "failed to fetch material", "error", err, "material_id", in.MaterialID)
		return nil, apperror.NewInternal()
	}

	if m.GroupID() != in.GroupID {
		return nil, apperror.NewNotFound(domainMaterial.ErrCodeMaterialNotFound, "material not found")
	}

	if !m.CanBeEditedBy(shared.RestoreUserID(in.CurrentUser.ID), in.CurrentUser.IsAdmin()) {
		return nil, apperror.NewForbidden(ErrCodeNotMaterialAuthor, "only the material author can update this material")
	}

	var fieldErrs []apperror.FieldError

	var title *domainMaterial.Title
	if in.Title != nil {
		t, tErr := domainMaterial.NewTitle(*in.Title)
		if err := apperror.AccumulateFieldErrors(tErr, &fieldErrs); err != nil {
			return nil, err
		}
		if tErr == nil {
			title = &t
		}
	}

	var content *domainMaterial.Content
	if in.Content != nil {
		c, cErr := domainMaterial.NewContent(*in.Content)
		if err := apperror.AccumulateFieldErrors(cErr, &fieldErrs); err != nil {
			return nil, err
		}
		if cErr == nil {
			content = &c
		}
	}

	var tags *domainMaterial.Tags
	if in.Tags != nil {
		t, tErr := domainMaterial.NewTags(*in.Tags)
		if err := apperror.AccumulateFieldErrors(tErr, &fieldErrs); err != nil {
			return nil, err
		}
		if tErr == nil {
			tags = &t
		}
	}

	if len(fieldErrs) > 0 {
		return nil, apperror.NewValidation(fieldErrs)
	}

	m.UpdateMetadata(title, content, tags)

	if err := uc.repo.Save(ctx, m); err != nil {
		slog.ErrorContext(ctx, "failed to save updated material", "error", err, "material_id", in.MaterialID)
		return nil, apperror.NewInternal()
	}

	return &UpdateMaterialOutput{Material: toMaterialData(m)}, nil
}
