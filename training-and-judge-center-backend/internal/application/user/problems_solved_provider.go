package user

import "context"

// ProblemsSolvedProvider computes the number of unique problems the user has solved.
type ProblemsSolvedProvider interface {
	// GetProblemsSolved returns the count of unique problems with at least
	// one ACCEPTED submission by the user.
	GetProblemsSolved(ctx context.Context, userID string) (int, error)
}
