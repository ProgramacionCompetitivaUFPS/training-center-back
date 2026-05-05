package problem

import (
	"context"
	"log/slog"
	"time"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RemoveModifierInput struct {
	Slug        string
	UserID      string
	CurrentUser appshared.CurrentUser
}

type RemoveModifierUseCase struct {
	repo problem.Repository
}

func NewRemoveModifierUseCase(repo problem.Repository) *RemoveModifierUseCase {
	return &RemoveModifierUseCase{repo: repo}
}

func (uc *RemoveModifierUseCase) Execute(ctx context.Context, input RemoveModifierInput) (struct{}, error) {
	slug, err := problem.NewSlug(input.Slug)
	if err != nil {
		return struct{}{}, err
	}

	p, err := uc.repo.FindBySlug(ctx, slug)
	if err != nil {
		return struct{}{}, err
	}

	isAuthor := p.AuthorID() == shared.RestoreUserID(input.CurrentUser.ID)
	isAdmin := input.CurrentUser.IsAdmin()

	if !isAuthor && !isAdmin {
		return struct{}{}, apperror.NewForbidden(apperror.ErrCodeForbidden, "Only the author or Admin can remove modifiers")
	}

	modifierID, err := shared.NewUserID(input.UserID)
	if err != nil {
		return struct{}{}, err
	}

	if err := p.RemoveModifier(modifierID, time.Now()); err != nil {
		return struct{}{}, err
	}

	if err := uc.repo.Save(ctx, p); err != nil {
		slog.ErrorContext(ctx, "failed to save problem after removing modifier", "error", err, "slug", p.Slug().String())
		return struct{}{}, apperror.NewInternal()
	}

	return struct{}{}, nil
}
