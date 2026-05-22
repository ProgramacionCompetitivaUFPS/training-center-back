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

type UnpublishProblemInput struct {
	Slug        string
	CurrentUser appshared.CurrentUser
}

type UnpublishProblemUseCase struct {
	repo problem.Repository
}

func NewUnpublishProblemUseCase(repo problem.Repository) *UnpublishProblemUseCase {
	return &UnpublishProblemUseCase{repo: repo}
}

type UnpublishProblemOutput struct {
	Problem ProblemDTO
}

func (uc *UnpublishProblemUseCase) Execute(ctx context.Context, in UnpublishProblemInput) (*UnpublishProblemOutput, error) {
	slug, err := problem.NewSlug(in.Slug)
	if err != nil {
		return nil, err
	}

	p, err := uc.repo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	viewerID := shared.RestoreUserID(in.CurrentUser.ID)
	isAdmin := in.CurrentUser.IsAdmin()
	if !p.CanBeEditedBy(viewerID, isAdmin) {
		return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions, "Only the problem author, Admin, or assigned modifiers can unpublish this problem")
	}

	now := time.Now()
	if err := p.Unpublish(now); err != nil {
		return nil, err
	}

	if err := uc.repo.Save(ctx, p); err != nil {
		slog.ErrorContext(ctx, "failed to save problem after unpublish", "error", err, "slug", p.Slug().String())
		return nil, apperror.NewInternal()
	}

	return &UnpublishProblemOutput{Problem: problemToDTO(p)}, nil
}
