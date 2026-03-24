package problem

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ListModifiersUseCase struct {
	repo problem.Repository
}

func NewListModifiersUseCase(repo problem.Repository) *ListModifiersUseCase {
	return &ListModifiersUseCase{repo: repo}
}

func (uc *ListModifiersUseCase) Execute(ctx context.Context, slugStr string, currentUser user.CurrentUser) ([]string, error) {
	slug, err := problem.NewSlug(slugStr)
	if err != nil {
		return nil, err
	}

	p, err := uc.repo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "Problem not found")
	}

	if !p.CanBeEditedBy(currentUser.ID, currentUser.Role) {
		return nil, apperror.NewForbidden(apperror.ErrCodeForbidden, "Only the problem author, Admin, or assigned modifiers can view modifiers")
	}

	return p.ModifierIDs, nil
}
