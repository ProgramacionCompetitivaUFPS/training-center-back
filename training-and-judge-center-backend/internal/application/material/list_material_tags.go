package material

import (
	"context"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ListMaterialTagsInput struct {
	CurrentUser appshared.CurrentUser
	GroupID     string
}

type ListMaterialTagsOutput struct {
	Tags []string
}

type ListMaterialTagsUseCase struct {
	repo            domainMaterial.Repository
	groupVisibility GroupVisibilityProvider
	memberProvider  GroupMemberProvider
}

func NewListMaterialTagsUseCase(
	repo domainMaterial.Repository,
	groupVisibility GroupVisibilityProvider,
	memberProvider GroupMemberProvider,
) *ListMaterialTagsUseCase {
	return &ListMaterialTagsUseCase{
		repo:            repo,
		groupVisibility: groupVisibility,
		memberProvider:  memberProvider,
	}
}

// Execute returns the distinct tags actually used by materials in the group, so the frontend
// can offer real filter options instead of a hardcoded guess. Non-leads only see tags that
// appear on at least one PUBLISHED material, mirroring the status visibility rule in List.
func (uc *ListMaterialTagsUseCase) Execute(ctx context.Context, in ListMaterialTagsInput) (*ListMaterialTagsOutput, error) {
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

	var statuses []domainMaterial.Status
	if !in.CurrentUser.IsAdmin() {
		isLead, leadErr := uc.memberProvider.IsLeadOfGroup(ctx, in.CurrentUser.ID, in.GroupID)
		if leadErr != nil {
			return nil, leadErr
		}
		if !isLead {
			statuses = []domainMaterial.Status{domainMaterial.NewStatusPublished()}
		}
	}

	tags, err := uc.repo.DistinctTags(ctx, in.GroupID, statuses)
	if err != nil {
		return nil, err
	}

	return &ListMaterialTagsOutput{Tags: tags}, nil
}
