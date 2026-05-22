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
	Nicknames []string
}

type ListModifiersUseCase struct {
	repo problem.Repository
}

func NewListModifiersUseCase(repo problem.Repository) *ListModifiersUseCase {
	return &ListModifiersUseCase{repo: repo}
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

	nicknames := make([]string, len(p.ModifierIDs()))
	for i, id := range p.ModifierIDs() {
		nicknames[i] = id.Value()
	}
	return &ListModifiersOutput{Nicknames: nicknames}, nil
}
