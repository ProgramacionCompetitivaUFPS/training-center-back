package problem

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// StatsProvider implements application/user.TopicStatsProvider.
type StatsProvider struct {
	db *pgxpool.Pool
}

func NewStatsProvider(db *pgxpool.Pool) *StatsProvider {
	return &StatsProvider{db: db}
}

// GetTopicBreakdown returns, for each tag on a problem the user solved, the
// count of unique problems solved carrying that tag, ordered by count descending.
func (p *StatsProvider) GetTopicBreakdown(ctx context.Context, userID string) ([]appuser.TopicStat, error) {
	rows, err := p.db.Query(ctx, `
		SELECT tag, COUNT(DISTINCT p.id)::INTEGER AS solved
		FROM problems p, unnest(p.tags) AS tag
		WHERE p.id IN (
			SELECT DISTINCT problem_id FROM submissions
			WHERE user_id = $1 AND status = 'ACCEPTED'
		)
		GROUP BY tag
		ORDER BY solved DESC`,
		userID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "profile stats: failed to query topic breakdown", "user_id", userID, "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	var result []appuser.TopicStat
	for rows.Next() {
		var t appuser.TopicStat
		if err := rows.Scan(&t.Tag, &t.Solved); err != nil {
			slog.ErrorContext(ctx, "profile stats: failed to scan topic stat row", "error", err)
			return nil, apperror.NewInternal()
		}
		result = append(result, t)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "profile stats: topic breakdown rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}
