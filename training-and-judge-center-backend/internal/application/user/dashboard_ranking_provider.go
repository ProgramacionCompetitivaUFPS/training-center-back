package user

import "context"

// DashboardRankingProvider computes the user's global ranking stats.
type DashboardRankingProvider interface {
	// GetUserStats returns:
	//   - problemsSolved: count of unique problems with at least one ACCEPTED submission
	//   - position: rank among all users by problemsSolved (1-based); nil if 0 solved
	//   - totalUsers: count of active users (for ranking context)
	GetUserStats(ctx context.Context, userID string) (problemsSolved int, position *int, totalUsers int, err error)
}
