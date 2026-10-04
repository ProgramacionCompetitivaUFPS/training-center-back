package submission

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainsubmission "github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RejudgeSubmissionInput struct {
	SubmissionID string
	CurrentUser  appshared.CurrentUser
	Now          time.Time
}

type RejudgeSubmissionOutput struct {
	SubmissionID    string
	ProblemSlug     string
	PreviousVerdict string
}

type RejudgeSubmissionUseCase struct {
	submissionRepo       domainsubmission.Repository
	judgingProvider      ProblemJudgingProvider
	contestTimesProvider ContestTimesProvider
	rejudger             SingleSubmissionRejudger
}

func NewRejudgeSubmissionUseCase(
	submissionRepo domainsubmission.Repository,
	judgingProvider ProblemJudgingProvider,
	contestTimesProvider ContestTimesProvider,
	rejudger SingleSubmissionRejudger,
) *RejudgeSubmissionUseCase {
	return &RejudgeSubmissionUseCase{
		submissionRepo:       submissionRepo,
		judgingProvider:      judgingProvider,
		contestTimesProvider: contestTimesProvider,
		rejudger:             rejudger,
	}
}

func (uc *RejudgeSubmissionUseCase) Execute(ctx context.Context, in RejudgeSubmissionInput) (*RejudgeSubmissionOutput, error) {
	sub, err := uc.submissionRepo.FindByID(ctx, in.SubmissionID)
	if err != nil {
		return nil, err
	}

	isAdmin := in.CurrentUser.IsAdmin()

	if !isAdmin && sub.UserID().String() != in.CurrentUser.ID {
		return nil, apperror.NewForbidden(domainsubmission.ErrCodeAccessDenied, "you can only rejudge your own submissions")
	}

	if !isAdmin && sub.ContestID() != nil {
		startTime, endTime, err := uc.contestTimesProvider.GetContestTimes(ctx, *sub.ContestID())
		if err != nil {
			return nil, err
		}
		contestActive := in.Now.After(startTime) && in.Now.Before(endTime)
		submittedDuringContest := sub.SubmittedAt().After(startTime) && sub.SubmittedAt().Before(endTime)
		if contestActive && submittedDuringContest {
			return nil, apperror.NewForbidden(ErrCodeCannotRejudgeInActiveContest,
				"cannot rejudge own submissions in active contests")
		}
	}

	if !isAdmin {
		judgingUpdatedAt, err := uc.judgingProvider.GetJudgingUpdatedAt(ctx, sub.ProblemID())
		if err != nil {
			return nil, err
		}
		if judgingUpdatedAt == nil || !sub.SubmittedAt().Before(*judgingUpdatedAt) {
			return nil, apperror.NewBadRequest(ErrCodeNoRejudgeNeeded,
				"this submission does not need rejudging; judging components have not been updated since submission")
		}
	}

	// A deleted problem leaves its submissions without a problem_id.
	if sub.ProblemID() == "" {
		return nil, apperror.NewNotFound(domainsubmission.ErrCodeProblemNotFound,
			"the problem of this submission no longer exists, so it cannot be rejudged")
	}

	published, err := uc.judgingProvider.IsPublished(ctx, sub.ProblemID())
	if err != nil {
		return nil, err
	}
	if !published {
		return nil, apperror.NewBadRequest(domainsubmission.ErrCodeProblemNotPublished,
			"only submissions of PUBLISHED problems can be rejudged")
	}

	previousVerdict := sub.Status().String()

	if err := uc.rejudger.RejudgeByID(ctx, sub.ID(), sub.ProblemID(), sub.UserID().String(), sub.ContestID(), sub.Language().String(), in.Now); err != nil {
		return nil, err
	}

	return &RejudgeSubmissionOutput{
		SubmissionID:    sub.ID(),
		ProblemSlug:     sub.ProblemSlug(),
		PreviousVerdict: previousVerdict,
	}, nil
}
