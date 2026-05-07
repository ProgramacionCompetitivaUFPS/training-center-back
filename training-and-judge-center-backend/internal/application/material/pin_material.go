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

type PinMaterialInput struct {
	CurrentUser appshared.CurrentUser
	GroupID     string
	MaterialID  string
}

type PinMaterialOutput struct {
	Material MaterialData
}

type PinMaterial struct {
	repo           domainMaterial.Repository
	groupProvider  GroupProvider
	memberProvider GroupMemberProvider
	authorProvider AuthorProvider
}

func NewPinMaterial(
	repo domainMaterial.Repository,
	groupProvider GroupProvider,
	memberProvider GroupMemberProvider,
	authorProvider AuthorProvider,
) *PinMaterial {
	return &PinMaterial{
		repo:           repo,
		groupProvider:  groupProvider,
		memberProvider: memberProvider,
		authorProvider: authorProvider,
	}
}

func (uc *PinMaterial) Execute(ctx context.Context, in PinMaterialInput) (*PinMaterialOutput, error) {
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
		return nil, apperror.NewForbidden(ErrCodeInsufficientPerms, "only the material author, a group lead, or an admin can pin materials")
	}

	// Idempotent: already-pinned materials return 200 with current state.
	if !m.Pinned() {
		now := time.Now()
		if err := m.Pin(now); err != nil {
			return nil, err
		}
		if err := uc.repo.Save(ctx, m); err != nil {
			slog.ErrorContext(ctx, "failed to save pinned material", "error", err, "material_id", in.MaterialID)
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

	return &PinMaterialOutput{Material: data}, nil
}
