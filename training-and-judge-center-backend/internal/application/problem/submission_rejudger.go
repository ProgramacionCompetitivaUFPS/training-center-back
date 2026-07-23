package problem

import (
	"context"
	"time"
)

type SubmissionRejudgeInfo struct {
	ID        string
	UserID    string
	ContestID *string
	Language  string
}

// SubmissionRejudger lists submissions eligible for rejudging and re-enqueues them in batch.
type SubmissionRejudger interface {
	ListByProblemBefore(ctx context.Context, problemID string, before time.Time) ([]SubmissionRejudgeInfo, error)
	ListByProblemAndContestBefore(ctx context.Context, problemID, contestID string, before time.Time) ([]SubmissionRejudgeInfo, error)
	// RejudgeBatch publishes all submissions to the judge queue and resets their status in one
	// batch UPDATE. Partial publish failures are logged and skipped; the returned count reflects
	// only successfully enqueued submissions.
	RejudgeBatch(ctx context.Context, subs []SubmissionRejudgeInfo, problemID string, now time.Time) (queued int, err error)
}
