package submission

import "context"

type ProblemDisplay struct {
	Slug  string
	Title string
}

type ProblemDisplayProvider interface {
	GetProblemByID(ctx context.Context, problemID string) (*ProblemDisplay, error)
	// GetProblemsByIDs batches what would otherwise be one GetProblemByID call per row in a
	// list — missing IDs (deleted problems) are simply absent from the map, not an error.
	GetProblemsByIDs(ctx context.Context, problemIDs []string) (map[string]*ProblemDisplay, error)
}
