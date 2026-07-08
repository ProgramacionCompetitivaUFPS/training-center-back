package problem

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RejudgeContestSubmissionsInput struct {
	ContestID   string
	Slug        string
	GroupID     string
	CurrentUser appshared.CurrentUser
	Now         time.Time
}

type RejudgeContestSubmissionsOutput struct {
	ContestID         string
	ProblemSlug       string
	SubmissionsQueued int
	ContestStatus     string
	StandingWillUpdate bool
}

type RejudgeContestSubmissionsUseCase struct {
	repo            problem.Repository
	rejudger        SubmissionRejudger
	contestProvider ContestRejudgeProvider
}

func NewRejudgeContestSubmissionsUseCase(
	repo problem.Repository,
	rejudger SubmissionRejudger,
	contestProvider ContestRejudgeProvider,
) *RejudgeContestSubmissionsUseCase {
	return &RejudgeContestSubmissionsUseCase{
		repo:            repo,
		rejudger:        rejudger,
		contestProvider: contestProvider,
	}
}

func (uc *RejudgeContestSubmissionsUseCase) Execute(ctx context.Context, in RejudgeContestSubmissionsInput) (*RejudgeContestSubmissionsOutput, error) {
	contest, err := uc.contestProvider.GetContestForRejudge(ctx, in.ContestID)
	if err != nil {
		return nil, err
	}

	if contest.GroupID == nil || *contest.GroupID != in.GroupID {
		return nil, apperror.NewNotFound(ErrCodeContestNotFound, "contest not found")
	}

	if !in.Now.After(contest.StartTime) || !in.Now.Before(contest.EndTime) {
		return nil, apperror.NewBadRequest(ErrCodeContestNotActive, "rejudge is only allowed during active contests")
	}

	userID := in.CurrentUser.ID
	if contest.OwnerID != userID {
		if contest.GroupID == nil {
			return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions,
				"only the contest owner or Leads of the group can rejudge submissions")
		}
		isLead, err := uc.contestProvider.IsLeadOfGroup(ctx, userID, *contest.GroupID)
		if err != nil {
			return nil, err
		}
		if !isLead {
			return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions,
				"only the contest owner or Leads of the group can rejudge submissions")
		}
	}

	slug, err := problem.NewSlug(in.Slug)
	if err != nil {
		return nil, err
	}

	p, err := uc.repo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	inContest, err := uc.contestProvider.IsProblemInContest(ctx, in.ContestID, p.ID())
	if err != nil {
		return nil, err
	}
	if !inContest {
		return nil, apperror.NewBadRequest(ErrCodeProblemNotInContest, "the specified problem is not part of this contest")
	}

	if p.JudgingUpdatedAt() == nil {
		return nil, apperror.NewBadRequest(ErrCodeNoSubmissionsToRejudge,
			"no judging updates have been recorded for this problem; nothing to rejudge")
	}

	submissions, err := uc.rejudger.ListByProblemAndContestBefore(ctx, p.ID(), in.ContestID, *p.JudgingUpdatedAt())
	if err != nil {
		return nil, err
	}
	if len(submissions) == 0 {
		return nil, apperror.NewBadRequest(ErrCodeNoSubmissionsToRejudge,
			"no contest submissions predate the last judging update; nothing to rejudge")
	}

	queued, err := uc.rejudger.RejudgeBatch(ctx, submissions, p.ID(), in.Now)
	if err != nil {
		return nil, err
	}

	return &RejudgeContestSubmissionsOutput{
		ContestID:          in.ContestID,
		ProblemSlug:        p.Slug().String(),
		SubmissionsQueued:  queued,
		ContestStatus:      contestStatusString(in.Now, contest.StartTime, contest.EndTime),
		StandingWillUpdate: queued > 0,
	}, nil
}
