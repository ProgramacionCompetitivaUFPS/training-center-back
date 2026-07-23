package problem

import (
	"context"

	"time"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type AddModifierInput struct {
	Slug         string
	UserNickname string
	CurrentUser  appshared.CurrentUser
}

type AddModifierUseCase struct {
	repo         problem.Repository
	userProvider UserProvider
}

func NewAddModifierUseCase(repo problem.Repository, userProvider UserProvider) *AddModifierUseCase {
	return &AddModifierUseCase{
		repo:         repo,
		userProvider: userProvider,
	}
}

func (uc *AddModifierUseCase) Execute(ctx context.Context, input AddModifierInput) error {
	slug, err := problem.NewSlug(input.Slug)
	if err != nil {
		return err
	}

	p, err := uc.repo.FindBySlug(ctx, slug)
	if err != nil {
		return err
	}

	isAuthor := p.AuthorID() == shared.RestoreUserID(input.CurrentUser.ID)
	isAdmin := input.CurrentUser.IsAdmin()

	if !isAuthor && !isAdmin {
		return apperror.NewForbidden(ErrCodeInsufficientPermissions, "Only the author or Admin can add modifiers")
	}

	modifierID, err := resolveModifierID(ctx, uc.userProvider, input.UserNickname)
	if err != nil {
		return err
	}

	now := time.Now()
	if err := p.AddModifier(modifierID, now); err != nil {
		return err
	}

	if err := uc.repo.Save(ctx, p); err != nil {
		return err
	}

	return nil
}
