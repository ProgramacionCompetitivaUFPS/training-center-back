package contest

import "context"

// ProblemInfo holds contest-relevant fields when validating slugs for add/update.
// CanAdd is resolved by the adapter: PUBLIC || isAdmin || isAuthor || isModifier.
type ProblemInfo struct {
	ID          string
	Slug        string
	Title       string
	IsPublished bool
	CanAdd      bool
}

// ProblemBasicInfo is used to enrich problem IDs with slug+title for responses.
type ProblemBasicInfo struct {
	ID    string
	Slug  string
	Title string
}

// ProblemWithLimits extends basic info with judging limits for the contest detail response.
type ProblemWithLimits struct {
	ID          string
	Slug        string
	Title       string
	TimeLimit   int
	MemoryLimit int
}

type ProblemProvider interface {
	// FindBySlugs returns a map keyed by slug. Missing slugs are absent from the map.
	FindBySlugs(ctx context.Context, slugs []string, callerID string, isAdmin bool) (map[string]*ProblemInfo, error)
	// FindByIDs returns a map keyed by problem ID. Used to enrich existing contest problems for responses.
	FindByIDs(ctx context.Context, ids []string) (map[string]*ProblemBasicInfo, error)
	// FindByIDsWithLimits returns problem info including time_limit and memory_limit, keyed by problem ID.
	FindByIDsWithLimits(ctx context.Context, ids []string) (map[string]*ProblemWithLimits, error)
}
