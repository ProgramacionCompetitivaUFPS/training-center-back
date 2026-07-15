package user

import "context"

// RankingProvider computes the user's global ranking stats. Intentionally
// separate from ProblemsSolvedProvider — this data is used only by the
// profile stats endpoint, which is expected to compute more expensive,
// slow-changing statistics that the dashboard never needs.
type RankingProvider interface {
	// GetRanking returns:
	//   - problemsSolved: count of unique problems with at least one ACCEPTED submission
	//   - position: rank among all users by problemsSolved (1-based); nil if 0 solved
	//   - totalUsers: count of active users (for ranking context)
	GetRanking(ctx context.Context, userID string) (problemsSolved int, position *int, totalUsers int, err error)
}
