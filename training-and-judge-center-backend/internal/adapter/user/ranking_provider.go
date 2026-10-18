package user

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/errgroup"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RankingProvider struct {
	db infraPostgres.Querier
}

var _ appuser.RankingProvider = (*RankingProvider)(nil)
var _ appuser.RankingSource = (*RankingProvider)(nil)

func NewRankingProvider(db infraPostgres.Querier) *RankingProvider {
	return &RankingProvider{db: db}
}

// GetRanking computes the user's unique problems solved, their global rank
// (1-based position by problems solved), and the total number of active users.
// Same "solved" logic as ProblemsSolvedProvider.GetProblemsSolved — keep in sync.
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

// GetAllRankings is GetRanking's whole-platform counterpart, used only by
// CachedRankingProvider to rebuild the cache. Same accepted_pairs → user_solved shape
// as GetRanking, but returns every user's count in one pass instead of discarding all
// but one — this is the query whose cost CachedRankingProvider amortizes across however
// many profile views happen before the cache goes stale.
func (p *RankingProvider) GetAllRankings(ctx context.Context) (map[string]int, int, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	g, gctx := errgroup.WithContext(ctx)

	var rows pgx.Rows
	var queryErr error
	var totalUsers int

	g.Go(func() error {
		rows, queryErr = q.Query(gctx, `
			SELECT user_id, COUNT(DISTINCT problem_id)::INTEGER AS solved
			FROM submissions
			WHERE status = 'ACCEPTED'
			GROUP BY user_id`,
		)
		return queryErr
	})
	g.Go(func() error {
		return q.QueryRow(gctx, `SELECT COUNT(*)::INTEGER FROM users WHERE status = 'ACTIVE'`).Scan(&totalUsers)
	})

	if err := g.Wait(); err != nil {
		slog.ErrorContext(ctx, "profile stats: failed to rebuild platform ranking", "error", err)
		return nil, 0, apperror.NewInternal()
	}
	defer rows.Close()

	solved := make(map[string]int)
	for rows.Next() {
		var userID string
		var count int
		if err := rows.Scan(&userID, &count); err != nil {
			slog.ErrorContext(ctx, "profile stats: failed to scan ranking row", "error", err)
			return nil, 0, apperror.NewInternal()
		}
		solved[userID] = count
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "profile stats: error iterating ranking rows", "error", err)
		return nil, 0, apperror.NewInternal()
	}
	return solved, totalUsers, nil
}
