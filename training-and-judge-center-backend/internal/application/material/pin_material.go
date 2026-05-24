package material

import (
	"context"
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

type PinMaterialUseCase struct {
	repo           domainMaterial.Repository
	groupProvider  GroupProvider
	memberProvider GroupMemberProvider
	authorProvider AuthorProvider
}

func NewPinMaterialUseCase(
	repo domainMaterial.Repository,
	groupProvider GroupProvider,
	memberProvider GroupMemberProvider,
	authorProvider AuthorProvider,
) *PinMaterialUseCase {
	return &PinMaterialUseCase{
		repo:           repo,
		groupProvider:  groupProvider,
		memberProvider: memberProvider,
		authorProvider: authorProvider,
	}
}

func (uc *PinMaterialUseCase) Execute(ctx context.Context, in PinMaterialInput) (*PinMaterialOutput, error) {
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
	if m.GroupID() != in.GroupID {
		return nil, apperror.NewNotFound(domainMaterial.ErrCodeMaterialNotFound, "material not found")
	}

	isGroupLead := false
	if !in.CurrentUser.IsAdmin() {
		isGroupLead, err = uc.memberProvider.IsLeadOfGroup(ctx, in.CurrentUser.ID, in.GroupID)
		if err != nil {
			return nil, err
		}
	}

	if !m.CanModifyPinStateBy(shared.RestoreUserID(in.CurrentUser.ID), in.CurrentUser.IsAdmin(), isGroupLead) {
		return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions, "only the material author, a group lead, or an admin can pin materials")
	}

	// Idempotent: already-pinned materials return 200 with current state.
	if !m.Pinned() {
		now := time.Now()
		if err := m.Pin(now); err != nil {
			return nil, err
		}
		if err := uc.repo.Save(ctx, m); err != nil {
			return nil, err
		}
	}

	data := toMaterialData(m)
	displays, err := uc.authorProvider.GetDisplays(ctx, []string{m.AuthorID().Value()})
	if err == nil {
		if disp := displays[m.AuthorID().Value()]; disp != nil {
			data.Author = &AuthorDTO{Nickname: disp.Nickname, Name: disp.Name}
		}
	}

	return &PinMaterialOutput{Material: data}, nil
}
