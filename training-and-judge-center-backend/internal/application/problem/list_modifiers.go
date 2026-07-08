package problem

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ListModifiersInput struct {
	Slug        string
	CurrentUser appshared.CurrentUser
}

type ListModifiersOutput struct {
	Modifiers []ModifierDisplay
}

type ListModifiersUseCase struct {
	repo         problem.Repository
	userProvider UserProvider
}

func NewListModifiersUseCase(repo problem.Repository, userProvider UserProvider) *ListModifiersUseCase {
	return &ListModifiersUseCase{repo: repo, userProvider: userProvider}
}

func (uc *ListModifiersUseCase) Execute(ctx context.Context, input ListModifiersInput) (*ListModifiersOutput, error) {
	slug, err := problem.NewSlug(input.Slug)
	if err != nil {
		return nil, err
	}

	p, err := uc.repo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	if !p.CanBeEditedBy(shared.RestoreUserID(input.CurrentUser.ID), input.CurrentUser.IsAdmin()) {
		return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions, "Only the problem author, Admin, or assigned modifiers can view modifiers")
	}

	modifierIDs := make([]string, len(p.ModifierIDs()))
	for i, id := range p.ModifierIDs() {
		modifierIDs[i] = id.Value()
	}
	displays, err := uc.userProvider.GetDisplays(ctx, modifierIDs)
	if err != nil {
		return nil, err
	}
	modifiers := make([]ModifierDisplay, 0, len(modifierIDs))
	for _, id := range modifierIDs {
		if d, ok := displays[id]; ok {
			modifiers = append(modifiers, ModifierDisplay{Nickname: d.Nickname, Name: d.Name})
		}
	}
	return &ListModifiersOutput{Modifiers: modifiers}, nil
}
