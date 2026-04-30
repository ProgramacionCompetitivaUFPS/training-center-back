package material

import (
	"context"
	"log/slog"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type UnpublishMaterialInput struct {
	CurrentUser shared.CurrentUser
	GroupID     string
	MaterialID  string
}

type UnpublishMaterialOutput struct {
	Material MaterialData
}

type UnpublishMaterial struct {
	repo           domainMaterial.Repository
	groupProvider  GroupProvider
	authorProvider AuthorProvider
}

func NewUnpublishMaterial(
	repo domainMaterial.Repository,
	groupProvider GroupProvider,
	authorProvider AuthorProvider,
) *UnpublishMaterial {
	return &UnpublishMaterial{repo: repo, groupProvider: groupProvider, authorProvider: authorProvider}
}

func (uc *UnpublishMaterial) Execute(ctx context.Context, in UnpublishMaterialInput) (*UnpublishMaterialOutput, error) {
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
		return nil, err
	}
	if m.GroupID() != in.GroupID {
		return nil, apperror.NewNotFound(domainMaterial.ErrCodeMaterialNotFound, "material not found")
	}

	if !m.CanBeEditedBy(shared.RestoreUserID(in.CurrentUser.ID), in.CurrentUser.IsAdmin()) {
		return nil, apperror.NewForbidden(ErrCodeNotMaterialAuthor, "only the material author can unpublish this material")
	}

	if !m.Status().IsDraft() {
		if err := m.Unpublish(); err != nil {
			slog.ErrorContext(ctx, "unexpected error unpublishing material", "error", err, "material_id", in.MaterialID)
			return nil, apperror.NewInternal()
		}
		if err := uc.repo.Save(ctx, m); err != nil {
			slog.ErrorContext(ctx, "failed to save unpublished material", "error", err, "material_id", in.MaterialID)
			return nil, apperror.NewInternal()
		}
	}

	data := toMaterialData(m)
	displays, err := uc.authorProvider.GetDisplays(ctx, []string{m.AuthorID().Value()})
	if err != nil {
		slog.ErrorContext(ctx, "failed to resolve author display", "error", err, "author_id", m.AuthorID().Value())
		return nil, apperror.NewInternal()
	}
	data.Author = displays[m.AuthorID().Value()]

	return &UnpublishMaterialOutput{Material: data}, nil
}
