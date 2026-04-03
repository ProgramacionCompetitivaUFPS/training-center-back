package problem

import (
	"context"

	"log/slog"
	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type AddModifierInput struct {
	Slug        string
	UserID      string
	CurrentUser user.CurrentUser
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
		return apperror.NewNotFound(apperror.ErrCodeNotFound, "Problem not found")
	}

	isAuthor := p.AuthorID == problem.RestoreUserID(input.CurrentUser.ID)
	isAdmin := input.CurrentUser.Role == user.RoleAdmin

	if !isAuthor && !isAdmin {
		return apperror.NewForbidden(apperror.ErrCodeForbidden, "Only the author or Admin can add modifiers")
	}

	exists, err := uc.userProvider.ExistsByID(ctx, input.UserID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check user existence", "error", err, "user_id", input.UserID)
		return apperror.NewInternal()
	}
	if !exists {
		return apperror.NewNotFound(apperror.ErrCodeNotFound, "user not found")
	}

	if err := p.AddModifier(problem.RestoreUserID(input.UserID)); err != nil {
		return err
	}

	if err := uc.repo.Save(ctx, p); err != nil {
		slog.ErrorContext(ctx, "failed to save problem with new modifier", "error", err, "slug", p.Slug.String())
		return apperror.NewInternal()
	}

	return nil
}
