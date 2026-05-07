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

type AddModifierInput struct {
	Slug        string
	UserID      string
	CurrentUser appshared.CurrentUser
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

func (uc *AddModifierUseCase) Execute(ctx context.Context, input AddModifierInput) (struct{}, error) {
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
		return struct{}{}, apperror.NewForbidden(apperror.ErrCodeForbidden, "Only the author or Admin can add modifiers")
	}

	exists, err := uc.userProvider.ExistsByID(ctx, input.UserID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check user existence", "error", err, "user_id", input.UserID)
		return struct{}{}, apperror.NewInternal()
	}
	if !exists {
		return struct{}{}, apperror.NewNotFound(ErrCodeUserNotFound, "User not found")
	}

	modifierID, err := shared.NewUserID(input.UserID)
	if err != nil {
		return struct{}{}, err
	}

	now := time.Now()
	if err := p.AddModifier(modifierID, now); err != nil {
		return struct{}{}, err
	}

	if err := uc.repo.Save(ctx, p); err != nil {
		slog.ErrorContext(ctx, "failed to save problem with new modifier", "error", err, "slug", p.Slug().String())
		return struct{}{}, apperror.NewInternal()
	}

	return struct{}{}, nil
}
