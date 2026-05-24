package material

import (
	"context"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GetMaterialInput struct {
	CurrentUser appshared.CurrentUser
	GroupID     string
	MaterialID  string
}

type GetMaterialOutput struct {
	Material MaterialData
}

type GetMaterialUseCase struct {
	repo               domainMaterial.Repository
	groupVisibility    GroupVisibilityProvider
	memberProvider     GroupMemberProvider
	authorProvider     AuthorProvider
}

func NewGetMaterialUseCase(
	repo domainMaterial.Repository,
	groupVisibility GroupVisibilityProvider,
	memberProvider GroupMemberProvider,
	authorProvider AuthorProvider,
) *GetMaterialUseCase {
	return &GetMaterialUseCase{
		repo:            repo,
		groupVisibility: groupVisibility,
		memberProvider:  memberProvider,
		authorProvider:  authorProvider,
	}
}

func (uc *GetMaterialUseCase) Execute(ctx context.Context, in GetMaterialInput) (*GetMaterialOutput, error) {
	visibility, exists, err := uc.groupVisibility.FindVisibility(ctx, in.GroupID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, apperror.NewNotFound(ErrCodeGroupNotFound, "group not found")
	}

	if err := checkGroupAccess(ctx, uc.memberProvider, in.CurrentUser, in.GroupID, visibility); err != nil {
		return nil, err
	}

	m, err := uc.repo.FindByID(ctx, in.MaterialID)
	if err != nil {
		return nil, err
	}
	if m.GroupID() != in.GroupID {
		return nil, apperror.NewNotFound(domainMaterial.ErrCodeMaterialNotFound, "material not found")
	}

	if !in.CurrentUser.IsAdmin() {
		isLead, err := uc.memberProvider.IsLeadOfGroup(ctx, in.CurrentUser.ID, in.GroupID)
		if err != nil {
			return nil, err
		}
		if m.Status().IsDraft() && !isLead {
			return nil, apperror.NewNotFound(domainMaterial.ErrCodeMaterialNotFound, "material not found")
		}
	}

	data := toMaterialData(m)
	displays, err := uc.authorProvider.GetDisplays(ctx, []string{m.AuthorID().Value()})
	if err != nil {
		return nil, err
	}
	if disp := displays[m.AuthorID().Value()]; disp != nil {
		data.Author = &AuthorDTO{Nickname: disp.Nickname, Name: disp.Name}
	}

	return &GetMaterialOutput{Material: data}, nil
}

