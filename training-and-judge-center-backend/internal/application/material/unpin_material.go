package material

import (
	"context"
	"log/slog"
	"time"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/internal/domain/shared"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type UnpinMaterialInput struct {
	CurrentUser appshared.CurrentUser
	GroupID     string
	MaterialID  string
}

type UnpinMaterialOutput struct {
	Material MaterialData
}

type UnpinMaterialUseCase struct {
	repo           domainMaterial.Repository
	groupProvider  GroupProvider
	memberProvider GroupMemberProvider
	authorProvider AuthorProvider
}

func NewUnpinMaterialUseCase(
	repo domainMaterial.Repository,
	groupProvider GroupProvider,
	memberProvider GroupMemberProvider,
	authorProvider AuthorProvider,
) *UnpinMaterialUseCase {
	return &UnpinMaterialUseCase{
		repo:           repo,
		groupProvider:  groupProvider,
		memberProvider: memberProvider,
		authorProvider: authorProvider,
	}
}

func (uc *UnpinMaterialUseCase) Execute(ctx context.Context, in UnpinMaterialInput) (*UnpinMaterialOutput, error) {
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

	isGroupLead := false
	if !in.CurrentUser.IsAdmin() {
		isGroupLead, err = uc.memberProvider.IsLeadOfGroup(ctx, in.CurrentUser.ID, in.GroupID)
		if err != nil {
			slog.ErrorContext(ctx, "failed to check lead role", "error", err, "user_id", in.CurrentUser.ID, "group_id", in.GroupID)
			return nil, apperror.NewInternal()
		}
	}

	if !m.CanModifyPinStateBy(shared.RestoreUserID(in.CurrentUser.ID), in.CurrentUser.IsAdmin(), isGroupLead) {
		return nil, apperror.NewForbidden(ErrCodeInsufficientPerms, "only the material author, a group lead, or an admin can unpin materials")
	}

	// Idempotent: already-unpinned materials return 200 with current state.
	if m.Pinned() {
		now := time.Now()
		m.Unpin(now)
		if err := uc.repo.Save(ctx, m); err != nil {
			slog.ErrorContext(ctx, "failed to save unpinned material", "error", err, "material_id", in.MaterialID)
			return nil, apperror.NewInternal()
		}
	}

	data := toMaterialData(m)
	displays, err := uc.authorProvider.GetDisplays(ctx, []string{m.AuthorID().Value()})
	if err != nil {
		slog.WarnContext(ctx, "failed to resolve author display, returning without author info", "error", err, "author_id", m.AuthorID().Value())
	} else {
		data.Author = displays[m.AuthorID().Value()]
	}

	return &UnpinMaterialOutput{Material: data}, nil
}
