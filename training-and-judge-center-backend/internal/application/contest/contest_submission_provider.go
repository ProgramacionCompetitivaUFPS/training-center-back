package contest

import (
	"context"
	"time"
)

type ContestSubmissionFilters struct {
	ProblemSlug *string
	Nickname    *string
}

type RichSubmissionData struct {
	ID                  string
	ProblemID           string
	ProblemSlug         string
	ProblemTitle        string
	ProblemOrder        int
	UserID              string
	StandingID          string
	Nickname            string
	SubmitterName       string
	TeamID              *string
	TeamName            *string
	TeamMemberNicknames []string
	Language            string
	Status              string
	SubmittedAt         time.Time
	JudgedAt            *time.Time
	TimeMs              *int
	MemoryKb            *int
}

type ContestSubmissionProvider interface {
	ListByContest(ctx context.Context, contestID string, filters ContestSubmissionFilters) ([]RichSubmissionData, error)
}
