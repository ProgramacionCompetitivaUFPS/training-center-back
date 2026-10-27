package material

import (
	"context"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// authorSuggestionSampleSize mirrors tagSuggestionSampleSize's rationale: a bounded sample of
// the group's most recent materials, not an exhaustive list — the `author` filter on List
// still accepts any nickname directly, this is only for suggesting who to pick.
const authorSuggestionSampleSize = 50

type ListMaterialAuthorsInput struct {
	CurrentUser appshared.CurrentUser
	GroupID     string
}

type ListMaterialAuthorsOutput struct {
	Authors []AuthorDTO
}

type ListMaterialAuthorsUseCase struct {
	repo            domainMaterial.Repository
	groupVisibility GroupVisibilityProvider
	memberProvider  GroupMemberProvider
	authorProvider  AuthorProvider
}

func NewListMaterialAuthorsUseCase(
	repo domainMaterial.Repository,
	groupVisibility GroupVisibilityProvider,
	memberProvider GroupMemberProvider,
	authorProvider AuthorProvider,
) *ListMaterialAuthorsUseCase {
	return &ListMaterialAuthorsUseCase{
		repo:            repo,
		groupVisibility: groupVisibility,
		memberProvider:  memberProvider,
		authorProvider:  authorProvider,
	}
}

// Execute returns the authors of a sample of the group's most recent materials, so the frontend
// can suggest real nicknames to filter by instead of requiring the exact nickname blind. Non-leads
// only see authors of at least one PUBLISHED material, mirroring the visibility rule in List.
func (uc *ListMaterialAuthorsUseCase) Execute(ctx context.Context, in ListMaterialAuthorsInput) (*ListMaterialAuthorsOutput, error) {
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

	authorIDs, err := uc.repo.DistinctAuthorIDs(ctx, in.GroupID, statuses, authorSuggestionSampleSize)
	if err != nil {
		return nil, err
	}

	displays, err := uc.authorProvider.GetDisplays(ctx, authorIDs)
	if err != nil {
		return nil, err
	}

	authors := make([]AuthorDTO, 0, len(authorIDs))
	for _, id := range authorIDs {
		if disp := displays[id]; disp != nil {
			authors = append(authors, AuthorDTO{Nickname: disp.Nickname, Name: disp.Name})
		}
	}

	return &ListMaterialAuthorsOutput{Authors: authors}, nil
}
