package material

import (
	"context"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/internal/domain/shared"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type DeleteMaterialInput struct {
	CurrentUser appshared.CurrentUser
	GroupID     string
	MaterialID  string
}

type DeleteMaterialUseCase struct {
	repo          domainMaterial.Repository
	groupProvider GroupProvider
}

func NewDeleteMaterialUseCase(repo domainMaterial.Repository, groupProvider GroupProvider) *DeleteMaterialUseCase {
	return &DeleteMaterialUseCase{repo: repo, groupProvider: groupProvider}
}

func (uc *DeleteMaterialUseCase) Execute(ctx context.Context, in DeleteMaterialInput) error {
	exists, err := uc.groupProvider.Exists(ctx, in.GroupID)
	if err != nil {
		return err
	}
	if !exists {
		return apperror.NewNotFound(ErrCodeGroupNotFound, "group not found")
	}

	m, err := uc.repo.FindByID(ctx, in.MaterialID)
	if err != nil {
		return err
	}
	if m.GroupID() != in.GroupID {
		return apperror.NewNotFound(domainMaterial.ErrCodeMaterialNotFound, "material not found")
	}

	if !m.CanBeEditedBy(shared.RestoreUserID(in.CurrentUser.ID), in.CurrentUser.IsAdmin()) {
		return apperror.NewForbidden(ErrCodeNotMaterialAuthor, "only the material author can delete this material")
	}

	if err := uc.repo.Delete(ctx, in.MaterialID); err != nil {
		return err
	}

	return nil
}
