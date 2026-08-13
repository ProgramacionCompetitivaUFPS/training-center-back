package problem

import "context"

type ProblemStatusProvider interface {
	GetStatus(ctx context.Context, problemID string) (string, error)
}
