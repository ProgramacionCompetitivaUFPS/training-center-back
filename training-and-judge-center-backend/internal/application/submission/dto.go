package submission

import "time"

const maxPageLimit = 100

// SubmissionSummary is the list-view shape (no source code).
type SubmissionSummary struct {
	ID            string
	Status        string
	Visibility    string
	SubmittedAt   time.Time
	JudgedAt      *time.Time
	Problem       ProblemDisplay
	Contest       *ContestDisplay
	SubmittedBy   UserDisplay
	Language      string
	ExecutionTime *int
	MemoryKb      *int
}
