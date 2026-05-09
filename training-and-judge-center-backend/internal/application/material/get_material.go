package material

import (
	"context"
	"log/slog"

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
		slog.ErrorContext(ctx, "failed to find group visibility", "error", err, "group_id", in.GroupID)
		return nil, apperror.NewInternal()
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
			slog.ErrorContext(ctx, "failed to check lead role", "error", err, "user_id", in.CurrentUser.ID, "group_id", in.GroupID)
			return nil, apperror.NewInternal()
		}
		if m.Status().IsDraft() && !isLead {
			return nil, apperror.NewNotFound(domainMaterial.ErrCodeMaterialNotFound, "material not found")
		}
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

	return &GetMaterialOutput{Material: data}, nil
}

func checkGroupAccess(ctx context.Context, mp GroupMemberProvider, user appshared.CurrentUser, groupID string, visibility GroupVisibility) error {
	if visibility == GroupVisibilityVisible || user.IsAdmin() {
		return nil
	}
	if visibility != GroupVisibilityNotVisible {
		slog.WarnContext(ctx, "unrecognised group visibility value, treating as NOT_VISIBLE", "visibility", visibility, "group_id", groupID)
	}
	isMember, err := mp.IsMemberOfGroup(ctx, user.ID, groupID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check group membership", "error", err, "user_id", user.ID, "group_id", groupID)
		return apperror.NewInternal()
	}
	if !isMember {
		return apperror.NewForbidden(ErrCodeInsufficientPerms, "you do not have permission to view materials in this group")
	}
	return nil
}

