package submission

import "context"

type ProblemDisplay struct {
	Slug  string
	Title string
}

type ProblemDisplayProvider interface {
	GetProblemByID(ctx context.Context, problemID string) (*ProblemDisplay, error)
}
