package material

import (
	"context"
	"log/slog"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GetMaterialInput struct {
	CurrentUser shared.CurrentUser
	GroupID     string
	MaterialID  string
}

type GetMaterialOutput struct {
	Material MaterialData
}

type GetMaterial struct {
	repo               domainMaterial.Repository
	groupVisibility    GroupVisibilityProvider
	memberProvider     GroupMemberProvider
	authorProvider     AuthorProvider
}

func NewGetMaterial(
	repo domainMaterial.Repository,
	groupVisibility GroupVisibilityProvider,
	memberProvider GroupMemberProvider,
	authorProvider AuthorProvider,
) *GetMaterial {
	return &GetMaterial{
		repo:            repo,
		groupVisibility: groupVisibility,
		memberProvider:  memberProvider,
		authorProvider:  authorProvider,
	}
}

func (uc *GetMaterial) Execute(ctx context.Context, in GetMaterialInput) (*GetMaterialOutput, error) {
	visibility, exists, err := uc.groupVisibility.FindVisibility(ctx, in.GroupID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to find group visibility", "error", err, "group_id", in.GroupID)
		return nil, apperror.NewInternal()
	}
	if !exists {
		return nil, apperror.NewNotFound(ErrCodeGroupNotFound, "group not found")
	}

	if err := uc.checkGroupAccess(ctx, in.CurrentUser, in.GroupID, visibility); err != nil {
		return nil, err
	}

	m, err := uc.repo.FindByID(ctx, in.MaterialID)
	if err != nil {
		return nil, err
	}
	if m.GroupID() != in.GroupID {
		return nil, apperror.NewNotFound(domainMaterial.ErrCodeMaterialNotFound, "material not found")
	}

	isAdmin := in.CurrentUser.IsAdmin()
	if !isAdmin {
		isLead, err := uc.memberProvider.IsLeadOfGroup(ctx, in.CurrentUser.ID, in.GroupID)
		if err != nil {
			slog.ErrorContext(ctx, "failed to check lead role", "error", err)
			return nil, apperror.NewInternal()
		}
		if m.Status().IsDraft() && !isLead {
			return nil, apperror.NewNotFound(domainMaterial.ErrCodeMaterialNotFound, "material not found")
		}
	}

	data := toMaterialData(m)
	data.Author = uc.resolveAuthor(ctx, m.AuthorID().Value())

	return &GetMaterialOutput{Material: data}, nil
}

// checkGroupAccess returns 403 if the group is NOT_VISIBLE and the user is not a member/admin.
func (uc *GetMaterial) checkGroupAccess(ctx context.Context, user shared.CurrentUser, groupID, visibility string) error {
	if visibility == "VISIBLE" || user.IsAdmin() {
		return nil
	}
	isMember, err := uc.memberProvider.IsMemberOfGroup(ctx, user.ID, groupID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check group membership", "error", err)
		return apperror.NewInternal()
	}
	if !isMember {
		return apperror.NewForbidden(ErrCodeInsufficientPerms, "you do not have permission to view materials in this group")
	}
	return nil
}

func (uc *GetMaterial) resolveAuthor(ctx context.Context, authorID string) *AuthorData {
	displays, err := uc.authorProvider.GetDisplays(ctx, []string{authorID})
	if err != nil {
		slog.WarnContext(ctx, "failed to resolve author display", "author_id", authorID)
		return nil
	}
	d := displays[authorID]
	if d == nil {
		return nil
	}
	return &AuthorData{Nickname: d.Nickname, Name: d.Name}
}
