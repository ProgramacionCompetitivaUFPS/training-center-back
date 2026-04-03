package problem

import (
	"context"
	"fmt"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ChangeAccessibilityInput struct {
	Slug          string
	Accessibility string
	CurrentUser   user.CurrentUser
}

type ChangeAccessibilityOutput struct {
	Slug          string
	Accessibility string
	Status        string
	Message       string
}

type ChangeAccessibilityUseCase struct {
	repo problem.Repository
}

func NewChangeAccessibilityUseCase(repo problem.Repository) *ChangeAccessibilityUseCase {
	return &ChangeAccessibilityUseCase{repo: repo}
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

	viewerID := problem.RestoreUserID(in.CurrentUser.ID)
	isAdmin := in.CurrentUser.Role == user.RoleAdmin
	if !p.CanBeEditedBy(viewerID, isAdmin) {
		return nil, apperror.NewForbidden(apperror.ErrCodeForbidden, "Only the problem author, Admin, or assigned modifiers can change this problem's accessibility")
	}

	p.UpdateAccessibility(newAcc)

	if err := uc.repo.Save(ctx, p); err != nil {
		return nil, err
	}

	return &ChangeAccessibilityOutput{
		Slug:          p.Slug.String(),
		Accessibility: p.Accessibility.String(),
		Status:        p.Status.String(),
		Message:       fmt.Sprintf("Problem accessibility changed to %s", p.Accessibility.String()),
	}, nil
}
