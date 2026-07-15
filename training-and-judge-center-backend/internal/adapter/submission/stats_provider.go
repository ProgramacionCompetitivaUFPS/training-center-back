package submission

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// StatsProvider implements application/user.SubmissionStatsProvider.
type StatsProvider struct {
	db infraPostgres.Querier
}

func NewStatsProvider(db infraPostgres.Querier) *StatsProvider {
	return &StatsProvider{db: db}
}

func (p *StatsProvider) GetSubmissionCounts(ctx context.Context, userID string) (total int, accepted int, err error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	row := q.QueryRow(ctx, `
		SELECT COUNT(*)::INTEGER,
		       COUNT(*) FILTER (WHERE status = 'ACCEPTED')::INTEGER
		FROM submissions
		WHERE user_id = $1`,
		userID,
	)

	if scanErr := row.Scan(&total, &accepted); scanErr != nil {
		slog.ErrorContext(ctx, "profile stats: failed to query submission counts", "user_id", userID, "error", scanErr)
		return 0, 0, apperror.NewInternal()
	}
	return total, accepted, nil
}
