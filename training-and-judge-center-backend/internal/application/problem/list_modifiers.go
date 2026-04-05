package problem

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ListModifiersInput struct {
	Slug        string
	CurrentUser user.CurrentUser
}

type ListModifiersUseCase struct {
	repo problem.Repository
}

func NewListModifiersUseCase(repo problem.Repository) *ListModifiersUseCase {
	return &ListModifiersUseCase{repo: repo}
}

func (uc *ListModifiersUseCase) Execute(ctx context.Context, input ListModifiersInput) ([]string, error) {
	slug, err := problem.NewSlug(input.Slug)
	if err != nil {
		return nil, err
	}

	p, err := uc.repo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	if !p.CanBeEditedBy(problem.RestoreUserID(input.CurrentUser.ID), input.CurrentUser.Role == user.RoleAdmin) {
		return nil, apperror.NewForbidden(apperror.ErrCodeForbidden, "Only the problem author, Admin, or assigned modifiers can view modifiers")
	}

	modifiers := make([]string, len(p.ModifierIDs))
	for i, id := range p.ModifierIDs {
		modifiers[i] = id.Value()
	}
	return modifiers, nil
}
