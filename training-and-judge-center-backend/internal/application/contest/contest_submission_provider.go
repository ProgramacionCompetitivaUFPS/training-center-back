package contest

import (
	"context"
	"time"
)

type ContestSubmissionFilters struct {
	ProblemSlug *string
	Nickname    *string
	// phase: SubmittedAtMost keeps submitted_at <= t, SubmittedAfter keeps submitted_at > t
	SubmittedAtMost *time.Time
	SubmittedAfter  *time.Time
	// freeze: when FreezeAt is set, rows submitted at or after it are kept only
	// if they belong to ExceptStandingID
	FreezeAt         *time.Time
	ExceptStandingID string
	Limit            int
	Offset           int
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
	IsBlockly           bool
	Status              string
	SubmittedAt         time.Time
	JudgedAt            *time.Time
	TimeMs              *int
	MemoryKb            *int
	Visibility          string // "PRIVATE" | "PUBLIC"
}

type ContestSubmissionProvider interface {
	// ListByContest returns one page of matching submissions and the total match count.
	ListByContest(ctx context.Context, contestID string, filters ContestSubmissionFilters) ([]RichSubmissionData, int, error)
}
