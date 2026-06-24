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

// SubmissionRejudger lists submissions eligible for rejudging and re-enqueues each one.
type SubmissionRejudger interface {
	ListByProblemBefore(ctx context.Context, problemID string, before time.Time) ([]SubmissionRejudgeInfo, error)
	RejudgeOne(ctx context.Context, info SubmissionRejudgeInfo, problemID string, now time.Time) error
}
