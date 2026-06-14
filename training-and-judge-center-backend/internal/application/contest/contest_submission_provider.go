package contest

import (
	"context"
	"time"
)

// ContestSubmissionFilters contains the optional filters pushed down to the data source.
type ContestSubmissionFilters struct {
	ProblemSlug *string
	Nickname    *string
}

// RichSubmissionData is the full submission data needed to list contest submissions.
type RichSubmissionData struct {
	ID           string
	ProblemID    string
	ProblemSlug  string
	ProblemTitle string
	ProblemOrder int
	UserID       string
	Nickname     string
	Language     string
	Status       string
	SubmittedAt  time.Time
	JudgedAt     *time.Time
	TimeMs       *int
	MemoryKb     *int
}

// ContestSubmissionsProvider provides all submissions for a contest, ordered by
// submitted_at DESC, with problem and user details resolved.
type ContestSubmissionsProvider interface {
	ListByContest(ctx context.Context, contestID string, filters ContestSubmissionFilters) ([]RichSubmissionData, error)
}
