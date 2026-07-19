package problem

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type TopicStatsProvider struct {
	db infraPostgres.Querier
}

func NewTopicStatsProvider(db infraPostgres.Querier) *TopicStatsProvider {
	return &TopicStatsProvider{db: db}
}

// GetTopicBreakdown counts solved problems (ACCEPTED) per tag, ordered by count descending.
func (p *TopicStatsProvider) GetTopicBreakdown(ctx context.Context, userID string) ([]appuser.TopicStat, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	rows, err := q.Query(ctx, `
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
