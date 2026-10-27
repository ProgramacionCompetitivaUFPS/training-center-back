package submission

import "context"

type ProblemInfo struct {
	ID           string
	AuthorID     string
	Slug         string
	Title        string
	IsPublished  bool
	IsPublic     bool
	ModifierIDs  []string
	HasTestCases bool
}

type ProblemProvider interface {
	GetProblemBySlug(ctx context.Context, slug string) (*ProblemInfo, error)
}
