package problem

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type UnpublishProblemInput struct {
	Slug        string
	CurrentUser appshared.CurrentUser
}

type UnpublishProblemUseCase struct {
	repo           problem.Repository
	contestChecker ActiveContestChecker
}

func NewUnpublishProblemUseCase(repo problem.Repository, contestChecker ActiveContestChecker) *UnpublishProblemUseCase {
	return &UnpublishProblemUseCase{repo: repo, contestChecker: contestChecker}
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

	inActive, err := uc.contestChecker.IsProblemInActiveContest(ctx, p.ID())
	if err != nil {
		return nil, err
	}
	if inActive {
		return nil, apperror.NewConflict(problem.ErrCodeProblemInActiveContest, "Cannot unpublish a problem that is currently being used in an active contest")
	}

	now := time.Now()
	if err := p.Unpublish(now); err != nil {
		return nil, err
	}

	if err := uc.repo.Save(ctx, p); err != nil {
		return nil, err
	}

	return &UnpublishProblemOutput{Problem: problemToDTO(p)}, nil
}
