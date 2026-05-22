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

type ChangeAccessibilityInput struct {
	Slug          string
	Accessibility string
	CurrentUser   appshared.CurrentUser
}

type ChangeAccessibilityUseCase struct {
	repo problem.Repository
}

func NewChangeAccessibilityUseCase(repo problem.Repository) *ChangeAccessibilityUseCase {
	return &ChangeAccessibilityUseCase{repo: repo}
}

type ChangeAccessibilityOutput struct {
	Problem ProblemDTO
}

func (uc *ChangeAccessibilityUseCase) Execute(ctx context.Context, in ChangeAccessibilityInput) (*ChangeAccessibilityOutput, error) {
	newAcc, err := problem.NewAccessibility(in.Accessibility)
	if err != nil {
		return nil, err
	}

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
		return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions, "Only the problem author, Admin, or assigned modifiers can change this problem's accessibility")
	}

	now := time.Now()
	p.UpdateAccessibility(newAcc, now)

	if err := uc.repo.Save(ctx, p); err != nil {
		slog.ErrorContext(ctx, "failed to save problem after accessibility change", "error", err, "slug", p.Slug().String())
		return nil, apperror.NewInternal()
	}

	return &ChangeAccessibilityOutput{Problem: problemToDTO(p)}, nil
}
