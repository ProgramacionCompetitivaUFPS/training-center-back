package problem

import (
	"context"
	"time"
)

type ContestRejudgeInfo struct {
	ID        string
	OwnerID   string
	GroupID   *string
	StartTime time.Time
	EndTime   time.Time
}

type ContestRejudgeProvider interface {
	GetContestForRejudge(ctx context.Context, contestID string) (*ContestRejudgeInfo, error)
	IsProblemInContest(ctx context.Context, contestID, problemID string) (bool, error)
	IsLeadOfGroup(ctx context.Context, userID, groupID string) (bool, error)
}
