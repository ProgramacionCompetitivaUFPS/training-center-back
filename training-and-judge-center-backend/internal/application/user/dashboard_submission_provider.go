package user

import (
	"context"
	"time"
)

// DashboardSubmissionProvider provides submission data for the dashboard use case.
type DashboardSubmissionProvider interface {
	// GetRecentSubmissions returns the last N submissions by the user, most recent first.
	GetRecentSubmissions(ctx context.Context, userID string, limit int) ([]DashboardSubmission, error)
	// GetSubmissionDates returns the distinct UTC dates on which the user submitted,
	// sorted ascending. Used to compute current and maximum streak.
	GetSubmissionDates(ctx context.Context, userID string) ([]time.Time, error)
}

type DashboardSubmission struct {
	ID            string
	ProblemSlug   string
	ProblemTitle  string
	Verdict       string
	Language      string
	SubmittedAt   time.Time
	ExecutionTime *int // milliseconds, nil while pending/running
	MemoryKb      *int // KB, nil while pending/running
}
