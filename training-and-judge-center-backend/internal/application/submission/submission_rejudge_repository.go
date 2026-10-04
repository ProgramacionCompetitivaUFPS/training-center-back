package submission

import (
	"context"
	"time"
)

type ProblemJudgingProvider interface {
	GetJudgingUpdatedAt(ctx context.Context, problemID string) (*time.Time, error)
	// IsPublished is false for a problem that no longer exists.
	IsPublished(ctx context.Context, problemID string) (bool, error)
}

type ContestTimesProvider interface {
	GetContestTimes(ctx context.Context, contestID string) (startTime, endTime time.Time, err error)
}

type SingleSubmissionRejudger interface {
	RejudgeByID(ctx context.Context, submissionID, problemID, userID string, contestID *string, language string, now time.Time) error
}
