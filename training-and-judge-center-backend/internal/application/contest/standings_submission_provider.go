package contest

import (
	"context"
	"time"
)

// ContestSubmissionData is the minimal submission data needed to compute standings.
type ContestSubmissionData struct {
	UserID      string
	ProblemID   string
	Status      string // ACCEPTED | WRONG_ANSWER | TIME_LIMIT_EXCEEDED | MEMORY_LIMIT_EXCEEDED | RUNTIME_ERROR | ...
	SubmittedAt time.Time
}

// StandingsSubmissionProvider provides all judged submissions for a contest,
// ordered by submitted_at ASC, for standings calculation.
type StandingsSubmissionProvider interface {
	ListByContest(ctx context.Context, contestID string) ([]ContestSubmissionData, error)
}
