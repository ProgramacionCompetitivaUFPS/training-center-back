package problem

import (
	"context"
	"log/slog"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type AdminRejudgeSubmissionsInput struct {
	Slug        string
	ContestID   *string // nil = global (all contexts)
	CurrentUser appshared.CurrentUser
	Now         time.Time
}

type AdminRejudgeSubmissionsOutput struct {
	ProblemSlug       string
	SubmissionsQueued int
}

type AdminRejudgeSubmissionsUseCase struct {
	repo            problem.Repository
	rejudger        SubmissionRejudger
	contestProvider ContestRejudgeProvider
}

func NewAdminRejudgeSubmissionsUseCase(
	repo problem.Repository,
	rejudger SubmissionRejudger,
	contestProvider ContestRejudgeProvider,
) *AdminRejudgeSubmissionsUseCase {
	return &AdminRejudgeSubmissionsUseCase{
		repo:            repo,
		rejudger:        rejudger,
		contestProvider: contestProvider,
	}
}

func (uc *AdminRejudgeSubmissionsUseCase) Execute(ctx context.Context, in AdminRejudgeSubmissionsInput) (*AdminRejudgeSubmissionsOutput, error) {
	slug, err := problem.NewSlug(in.Slug)
	if err != nil {
		return nil, err
	}

	p, err := uc.repo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	if p.JudgingUpdatedAt() == nil {
		return &AdminRejudgeSubmissionsOutput{ProblemSlug: p.Slug().String(), SubmissionsQueued: 0}, nil
	}

	var submissions []SubmissionRejudgeInfo

	if in.ContestID != nil {
		if _, err := uc.contestProvider.GetContestForRejudge(ctx, *in.ContestID); err != nil {
			return nil, err
		}
		inContest, err := uc.contestProvider.IsProblemInContest(ctx, *in.ContestID, p.ID())
		if err != nil {
			return nil, err
		}
		if !inContest {
			return nil, apperror.NewBadRequest(ErrCodeProblemNotInContest, "the specified problem is not part of this contest")
		}
		submissions, err = uc.rejudger.ListByProblemAndContestBefore(ctx, p.ID(), *in.ContestID, *p.JudgingUpdatedAt())
		if err != nil {
			return nil, err
		}
	} else {
		submissions, err = uc.rejudger.ListByProblemBefore(ctx, p.ID(), *p.JudgingUpdatedAt())
		if err != nil {
			return nil, err
		}
	}

	queued := 0
	for _, sub := range submissions {
		if err := uc.rejudger.RejudgeOne(ctx, sub, p.ID(), in.Now); err != nil {
			slog.ErrorContext(ctx, "admin rejudge: failed to rejudge submission", "submission_id", sub.ID, "error", err)
			continue
		}
		queued++
	}

	return &AdminRejudgeSubmissionsOutput{
		ProblemSlug:       p.Slug().String(),
		SubmissionsQueued: queued,
	}, nil
}
