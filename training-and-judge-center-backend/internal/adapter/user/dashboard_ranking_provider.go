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

	row := q.QueryRow(ctx, `
		WITH my_solved AS (
			SELECT COUNT(DISTINCT problem_id)::INTEGER AS cnt
			FROM submissions
			WHERE user_id = $1 AND status = 'ACCEPTED'
		),
		better AS (
			SELECT COUNT(*)::INTEGER AS cnt
			FROM (
				SELECT user_id
				FROM submissions
				WHERE status = 'ACCEPTED'
				GROUP BY user_id
				HAVING COUNT(DISTINCT problem_id) > (SELECT cnt FROM my_solved)
			) sub
		)
		SELECT
			(SELECT cnt FROM my_solved) AS problems_solved,
			CASE
				WHEN (SELECT cnt FROM my_solved) = 0 THEN NULL
				ELSE (SELECT cnt FROM better) + 1
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
