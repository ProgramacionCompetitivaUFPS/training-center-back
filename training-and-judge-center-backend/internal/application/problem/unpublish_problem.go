package problem

import (
	"context"
	"log/slog"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type UnpublishProblemInput struct {
	Slug        string
	CurrentUser user.CurrentUser
}

type UnpublishProblemUseCase struct {
	repo problem.Repository
}

func NewUnpublishProblemUseCase(repo problem.Repository) *UnpublishProblemUseCase {
	return &UnpublishProblemUseCase{repo: repo}
}

func (uc *UnpublishProblemUseCase) Execute(ctx context.Context, in UnpublishProblemInput) (*problem.Problem, error) {
	slug, err := problem.NewSlug(in.Slug)
	if err != nil {
		return nil, err
	}

	p, err := uc.repo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	viewerID := problem.RestoreUserID(in.CurrentUser.ID)
	isAdmin := in.CurrentUser.Role == user.RoleAdmin
	if !p.CanBeEditedBy(viewerID, isAdmin) {
		return nil, apperror.NewForbidden(apperror.ErrCodeForbidden, "Only the problem author, Admin, or assigned modifiers can unpublish this problem")
	}

	if err := p.Unpublish(); err != nil {
		return nil, err
	}

	if err := uc.repo.Save(ctx, p); err != nil {
		slog.ErrorContext(ctx, "failed to save problem after unpublish", "error", err, "slug", p.Slug.String())
		return nil, apperror.NewInternal()
	}

	return p, nil
}
