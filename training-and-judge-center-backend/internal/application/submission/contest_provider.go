package submission

import "context"

type ContestDisplay struct {
	ID      string
	Name    string
	GroupID string
}

type ContestProvider interface {
	GetContestByID(ctx context.Context, contestID string) (*ContestDisplay, error)
	// GetProblemPosition returns the 1-based slot a problem occupies in a contest (the number
	// behind its A, B, C… letter), or nil when the problem is no longer part of that contest —
	// an edited contest can drop a problem that older submissions still point at.
	GetProblemPosition(ctx context.Context, contestID, problemID string) (*int, error)
	// GetContestsByIDs batches what would otherwise be one GetContestByID call per row in a
	// list. Unlike GetProblemsByIDs, every requested ID is expected to resolve — a contest_id
	// on a submission is never stale the way problem_id can be after a deletion — so the
	// caller treats a missing entry as an error, same as the single-row GetContestByID would.
	GetContestsByIDs(ctx context.Context, contestIDs []string) (map[string]*ContestDisplay, error)
}
