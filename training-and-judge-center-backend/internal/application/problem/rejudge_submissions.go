package problem

import (
	"context"
	"log/slog"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RejudgeSubmissionsInput struct {
	Slug        string
	CurrentUser appshared.CurrentUser
	Now         time.Time
}

type RejudgeSubmissionsOutput struct {
	ProblemSlug       string
	SubmissionsQueued int
}

type RejudgeSubmissionsUseCase struct {
	repo     problem.Repository
	rejudger SubmissionRejudger
}

func NewRejudgeSubmissionsUseCase(repo problem.Repository, rejudger SubmissionRejudger) *RejudgeSubmissionsUseCase {
	return &RejudgeSubmissionsUseCase{repo: repo, rejudger: rejudger}
}

func (uc *RejudgeSubmissionsUseCase) Execute(ctx context.Context, in RejudgeSubmissionsInput) (*RejudgeSubmissionsOutput, error) {
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
		return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions,
			"Only the problem author, Admin, or assigned modifiers can rejudge submissions")
	}

	if p.JudgingUpdatedAt() == nil {
		return nil, apperror.NewBadRequest(ErrCodeNoSubmissionsToRejudge,
			"no judging updates have been recorded for this problem; nothing to rejudge")
	}

	submissions, err := uc.rejudger.ListByProblemBefore(ctx, p.ID(), *p.JudgingUpdatedAt())
	if err != nil {
		return nil, err
	}
	if len(submissions) == 0 {
		return nil, apperror.NewBadRequest(ErrCodeNoSubmissionsToRejudge,
			"no submissions predate the last judging update; nothing to rejudge")
	}

	queued := 0
	for _, sub := range submissions {
		if err := uc.rejudger.RejudgeOne(ctx, sub, p.ID(), in.Now); err != nil {
			slog.ErrorContext(ctx, "rejudge: failed to rejudge submission", "submission_id", sub.ID, "error", err)
			continue
		}
		queued++
	}

	return &RejudgeSubmissionsOutput{
		ProblemSlug:       p.Slug().String(),
		SubmissionsQueued: queued,
	}, nil
}
