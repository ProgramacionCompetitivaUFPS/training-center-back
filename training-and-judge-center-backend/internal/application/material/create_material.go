package material

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/internal/domain/shared"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type CreateMaterialInput struct {
	CurrentUser appshared.CurrentUser
	GroupID     string
	Title       string
	Content     *string
	Tags        []string
}

type CreateMaterialOutput struct {
	Material MaterialData
}

type CreateMaterialUseCase struct {
	repo           domainMaterial.Repository
	groupProvider  GroupProvider
	memberProvider GroupMemberProvider
	authorProvider AuthorProvider
}

func NewCreateMaterialUseCase(
	repo domainMaterial.Repository,
	groupProvider GroupProvider,
	memberProvider GroupMemberProvider,
	authorProvider AuthorProvider,
) *CreateMaterialUseCase {
	return &CreateMaterialUseCase{
		repo:           repo,
		groupProvider:  groupProvider,
		memberProvider: memberProvider,
		authorProvider: authorProvider,
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
			return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions, "only group leads can create materials")
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

	newID := uuid.New().String()
	now := time.Now()
	m, err := domainMaterial.NewMaterial(
		newID,
		in.GroupID,
		shared.RestoreUserID(in.CurrentUser.ID),
		title,
		content,
		tags,
		now,
	)
	if err != nil {
		slog.ErrorContext(ctx, "unexpected error constructing material", "error", err, "group_id", in.GroupID)
		return nil, apperror.NewInternal()
	}

	if err := uc.repo.Save(ctx, m); err != nil {
		slog.ErrorContext(ctx, "failed to save material", "error", err, "material_id", m.ID())
		return nil, apperror.NewInternal()
	}

	data := toMaterialData(m)
	displays, err := uc.authorProvider.GetDisplays(ctx, []string{m.AuthorID().Value()})
	if err != nil {
		slog.ErrorContext(ctx, "failed to resolve author display", "error", err, "author_id", m.AuthorID().Value())
		return nil, apperror.NewInternal()
	}
	if disp := displays[m.AuthorID().Value()]; disp != nil {
		data.Author = &AuthorDTO{Nickname: disp.Nickname, Name: disp.Name}
	}

	return &CreateMaterialOutput{Material: data}, nil
}
