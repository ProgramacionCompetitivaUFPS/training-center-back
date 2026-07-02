package problem

import (
	"context"

	"time"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RemoveModifierInput struct {
	Slug         string
	UserNickname string
	CurrentUser  appshared.CurrentUser
}

type RemoveModifierUseCase struct {
	repo         problem.Repository
	userProvider UserProvider
}

func NewRemoveModifierUseCase(repo problem.Repository, userProvider UserProvider) *RemoveModifierUseCase {
	return &RemoveModifierUseCase{repo: repo, userProvider: userProvider}
}

func (uc *RemoveModifierUseCase) Execute(ctx context.Context, input RemoveModifierInput) error {
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
		return apperror.NewForbidden(ErrCodeInsufficientPermissions, "Only the author or Admin can remove modifiers")
	}

	modifierID, err := resolveModifierID(ctx, uc.userProvider, input.UserNickname)
	if err != nil {
		return err
	}

	now := time.Now()
	if err := p.RemoveModifier(modifierID, now); err != nil {
		return err
	}

	if err := uc.repo.Save(ctx, p); err != nil {
		return err
	}

	return nil
}
