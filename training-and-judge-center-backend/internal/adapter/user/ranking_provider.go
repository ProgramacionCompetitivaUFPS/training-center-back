package user

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RankingProvider struct {
	db infraPostgres.Querier
}

var _ appuser.RankingProvider = (*RankingProvider)(nil)

func NewRankingProvider(db infraPostgres.Querier) *RankingProvider {
	return &RankingProvider{db: db}
}

// GetRanking computes the user's unique problems solved, their global rank
// (1-based position by problems solved), and the total number of active users.
//
// The "unique problems solved" computation here duplicates the one in
// adapter/user/problems_solved_provider.go (GetProblemsSolved) — kept separate
// on purpose so /dashboard doesn't pay the cost of this ranking query. If the
// definition of "solved" ever changes, update both.
func (p *RankingProvider) GetRanking(ctx context.Context, userID string) (problemsSolved int, position *int, totalUsers int, err error) {
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
		slog.ErrorContext(ctx, "profile stats: failed to query ranking", "user_id", userID, "error", scanErr)
		return 0, nil, 0, apperror.NewInternal()
	}
	return problemsSolved, pos, totalUsers, nil
}
