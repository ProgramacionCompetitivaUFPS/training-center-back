package user

import "context"

// RankingProvider is kept separate from ProblemsSolvedProvider so /dashboard
// doesn't pay the cost of this ranking query on every load.
type RankingProvider interface {
	// GetRanking returns:
	//   - problemsSolved: count of unique problems with at least one ACCEPTED submission
	//   - position: rank among all users by problemsSolved (1-based); nil if 0 solved
	//   - totalUsers: count of active users (for ranking context)
	GetRanking(ctx context.Context, userID string) (problemsSolved int, position *int, totalUsers int, err error)
}
