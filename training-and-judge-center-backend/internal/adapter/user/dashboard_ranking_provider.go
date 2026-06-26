package user

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// DashboardRankingProvider implements application/user.DashboardRankingProvider.
type DashboardRankingProvider struct {
	db infraPostgres.Querier
}

func NewDashboardRankingProvider(db infraPostgres.Querier) *DashboardRankingProvider {
	return &DashboardRankingProvider{db: db}
}

// GetUserStats computes the user's unique problems solved, their global rank
// (1-based position by problems solved), and the total number of active users.
func (p *DashboardRankingProvider) GetUserStats(ctx context.Context, userID string) (problemsSolved int, position *int, totalUsers int, err error) {
	_ = appuser.DashboardRankingProvider(p) // compile-time interface check

	q := infraPostgres.GetQuerier(ctx, p.db)

	// accepted_pairs uses idx_submissions_accepted_user_problem (index-only scan).
	// user_solved derives per-user counts without COUNT(DISTINCT) on the raw table.
	row := q.QueryRow(ctx, `
		WITH accepted_pairs AS (
			SELECT DISTINCT user_id, problem_id
			FROM submissions
			WHERE status = 'ACCEPTED'
		),
		user_solved AS (
			SELECT user_id, COUNT(*)::INTEGER AS solved
			FROM accepted_pairs
			GROUP BY user_id
		)
		SELECT
			COALESCE((SELECT solved FROM user_solved WHERE user_id = $1), 0) AS problems_solved,
			CASE
				WHEN COALESCE((SELECT solved FROM user_solved WHERE user_id = $1), 0) = 0 THEN NULL
				ELSE (
					SELECT COUNT(*)::INTEGER + 1
					FROM user_solved
					WHERE solved > COALESCE((SELECT solved FROM user_solved WHERE user_id = $1), 0)
				)
			END AS position,
			(SELECT COUNT(*)::INTEGER FROM users WHERE status = 'ACTIVE') AS total_users`,
		userID,
	)

	var pos *int
	if scanErr := row.Scan(&problemsSolved, &pos, &totalUsers); scanErr != nil {
		slog.ErrorContext(ctx, "dashboard: failed to query user stats", "user_id", userID, "error", scanErr)
		return 0, nil, 0, apperror.NewInternal()
	}
	return problemsSolved, pos, totalUsers, nil
}
