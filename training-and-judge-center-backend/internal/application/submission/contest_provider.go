package submission

import "context"

type ContestDisplay struct {
	ID   string
	Name string
}

type ContestProvider interface {
	GetContestByID(ctx context.Context, contestID string) (*ContestDisplay, error)
	// GetContestsByIDs batches what would otherwise be one GetContestByID call per row in a
	// list. Unlike GetProblemsByIDs, every requested ID is expected to resolve — a contest_id
	// on a submission is never stale the way problem_id can be after a deletion — so the
	// caller treats a missing entry as an error, same as the single-row GetContestByID would.
	GetContestsByIDs(ctx context.Context, contestIDs []string) (map[string]*ContestDisplay, error)
}
