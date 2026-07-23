package problem

import (
	"context"
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
	ProblemSlug        string
	SubmissionsQueued  int
	ContestStatus      *string // non-nil only when contestID was specified
	StandingWillUpdate *bool   // non-nil only when contestID was specified
}

type AdminRejudgeSubmissionsUseCase struct {
	repo            problem.Repository
	rejudger        SubmissionRejudger
	contestProvider ContestRejudgeProvider
}

func contestStatusString(now, start, end time.Time) string {
	if now.Before(start) {
		return "SCHEDULED"
	}
	if !now.Before(end) {
		return "FINISHED"
	}
	return "ACTIVE"
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
		return nil, apperror.NewBadRequest(ErrCodeNoSubmissionsToRejudge,
			"no judging updates have been recorded for this problem; nothing to rejudge")
	}

	var submissions []SubmissionRejudgeInfo
	var contestStatus *string
	var standingWillUpdate *bool

	if in.ContestID != nil {
		contest, err := uc.contestProvider.GetContestForRejudge(ctx, *in.ContestID)
		if err != nil {
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
		status := contestStatusString(in.Now, contest.StartTime, contest.EndTime)
		contestStatus = &status
	} else {
		submissions, err = uc.rejudger.ListByProblemBefore(ctx, p.ID(), *p.JudgingUpdatedAt())
		if err != nil {
			return nil, err
		}
	}

	if len(submissions) == 0 {
		return nil, apperror.NewBadRequest(ErrCodeNoSubmissionsToRejudge,
			"no submissions predate the last judging update; nothing to rejudge")
	}

	queued, err := uc.rejudger.RejudgeBatch(ctx, submissions, p.ID(), in.Now)
	if err != nil {
		return nil, err
	}

	if contestStatus != nil {
		willUpdate := queued > 0 && *contestStatus == "ACTIVE"
		standingWillUpdate = &willUpdate
	}

	return &AdminRejudgeSubmissionsOutput{
		ProblemSlug:        p.Slug().String(),
		SubmissionsQueued:  queued,
		ContestStatus:      contestStatus,
		StandingWillUpdate: standingWillUpdate,
	}, nil
}
