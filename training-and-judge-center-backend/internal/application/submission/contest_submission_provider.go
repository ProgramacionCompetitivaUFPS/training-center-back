package submission

import (
	"context"
	"time"
)

type ContestSubmissionInfo struct {
	ID                string
	Name              string
	GroupID           string
	StartTime         time.Time
	EndTime           time.Time
	EnablePostContest bool
	ProblemIDs        []string
}

type ContestSubmissionProvider interface {
	GetContestForSubmission(ctx context.Context, groupID, contestID string) (*ContestSubmissionInfo, error)
}
