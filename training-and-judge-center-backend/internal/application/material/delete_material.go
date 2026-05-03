package material

import (
	"context"
	"errors"
	"log/slog"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type DeleteMaterialInput struct {
	CurrentUser shared.CurrentUser
	GroupID     string
	MaterialID  string
}

type DeleteMaterial struct {
	repo          domainMaterial.Repository
	groupProvider GroupProvider
}

func NewDeleteMaterial(repo domainMaterial.Repository, groupProvider GroupProvider) *DeleteMaterial {
	return &DeleteMaterial{repo: repo, groupProvider: groupProvider}
}

func (uc *DeleteMaterial) Execute(ctx context.Context, in DeleteMaterialInput) error {
	exists, err := uc.groupProvider.Exists(ctx, in.GroupID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check group existence", "error", err, "group_id", in.GroupID)
		return apperror.NewInternal()
	}
	if !exists {
		return apperror.NewNotFound(ErrCodeGroupNotFound, "group not found")
	}

	m, err := uc.repo.FindByID(ctx, in.MaterialID)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			return err
		}
		slog.ErrorContext(ctx, "failed to find material", "error", err, "material_id", in.MaterialID)
		return apperror.NewInternal()
	}
	if m.GroupID() != in.GroupID {
		return apperror.NewNotFound(domainMaterial.ErrCodeMaterialNotFound, "material not found")
	}

	if !m.CanBeEditedBy(shared.RestoreUserID(in.CurrentUser.ID), in.CurrentUser.IsAdmin()) {
		return apperror.NewForbidden(ErrCodeNotMaterialAuthor, "only the material author can delete this material")
	}

	if err := uc.repo.Delete(ctx, in.MaterialID); err != nil {
		slog.ErrorContext(ctx, "failed to delete material", "error", err, "material_id", in.MaterialID)
		return apperror.NewInternal()
	}

	return nil
}
