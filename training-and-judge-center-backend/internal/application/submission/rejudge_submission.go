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
		submittedDuringContest := sub.SubmittedAt().After(startTime) && !sub.SubmittedAt().After(endTime)
		if contestActive && submittedDuringContest {
			return nil, apperror.NewForbidden(domainsubmission.ErrCodeCannotRejudgeInActiveContest,
				"cannot rejudge own submissions in active contests")
		}
	}

	if !isAdmin {
		judgingUpdatedAt, err := uc.judgingProvider.GetJudgingUpdatedAt(ctx, sub.ProblemID())
		if err != nil {
			return nil, err
		}
		if judgingUpdatedAt == nil || !sub.SubmittedAt().Before(*judgingUpdatedAt) {
			return nil, apperror.NewBadRequest(domainsubmission.ErrCodeNoRejudgeNeeded,
				"this submission does not need rejudging; judging components have not been updated since submission")
		}
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
