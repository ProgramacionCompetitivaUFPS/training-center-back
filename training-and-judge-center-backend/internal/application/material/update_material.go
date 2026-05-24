package material

import (
	"context"
	"time"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/internal/domain/shared"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type UpdateMaterialInput struct {
	CurrentUser appshared.CurrentUser
	// GroupID scopes the operation: the material must belong to this group.
	// A material whose GroupID differs from this value is treated as not found.
	GroupID     string
	MaterialID  string
	Title       *string
	Content     *string
	Tags        *[]string // nil = no change; &[]string{} = clear all tags
}

type UpdateMaterialOutput struct {
	Material MaterialData
}

type UpdateMaterialUseCase struct {
	repo           domainMaterial.Repository
	groupProvider  GroupProvider
	authorProvider AuthorProvider
}

func NewUpdateMaterialUseCase(
	repo domainMaterial.Repository,
	groupProvider GroupProvider,
	authorProvider AuthorProvider,
) *UpdateMaterialUseCase {
	return &UpdateMaterialUseCase{repo: repo, groupProvider: groupProvider, authorProvider: authorProvider}
}

func (uc *UpdateMaterialUseCase) Execute(ctx context.Context, in UpdateMaterialInput) (*UpdateMaterialOutput, error) {
	exists, err := uc.groupProvider.Exists(ctx, in.GroupID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, apperror.NewNotFound(ErrCodeGroupNotFound, "group not found")
	}

	m, err := uc.repo.FindByID(ctx, in.MaterialID)
	if err != nil {
		return nil, err
	}

	// Treat material in another group as not found to avoid cross-group existence leak.
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

	var content **domainMaterial.Content
	if in.Content != nil {
		c, cErr := domainMaterial.NewContent(*in.Content)
		if err := apperror.AccumulateFieldErrors(cErr, &fieldErrs); err != nil {
			return nil, err
		}
		if cErr == nil {
			ptr := &c
			content = &ptr
		}
	}

	var tags **domainMaterial.Tags
	if in.Tags != nil {
		t, tErr := domainMaterial.NewTags(*in.Tags)
		if err := apperror.AccumulateFieldErrors(tErr, &fieldErrs); err != nil {
			return nil, err
		}
		if tErr == nil {
			ptr := &t
			tags = &ptr
		}
	}

	if len(fieldErrs) > 0 {
		return nil, apperror.NewValidation(fieldErrs)
	}

	now := time.Now()
	m.UpdateMetadata(title, content, tags, now)

	if err := uc.repo.Save(ctx, m); err != nil {
		return nil, err
	}

	data := toMaterialData(m)
	displays, err := uc.authorProvider.GetDisplays(ctx, []string{m.AuthorID().Value()})
	if err != nil {
		return nil, err
	}
	if disp := displays[m.AuthorID().Value()]; disp != nil {
		data.Author = &AuthorDTO{Nickname: disp.Nickname, Name: disp.Name}
	}

	return &UpdateMaterialOutput{Material: data}, nil
}
