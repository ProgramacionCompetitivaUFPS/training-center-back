package user

import "context"

type ProblemsSolvedProvider interface {
	// GetProblemsSolved returns the count of unique problems with at least
	// one ACCEPTED submission by the user.
	GetProblemsSolved(ctx context.Context, userID string) (int, error)
}
