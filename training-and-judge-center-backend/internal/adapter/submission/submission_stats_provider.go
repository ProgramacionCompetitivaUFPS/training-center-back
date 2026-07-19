package submission

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type SubmissionStatsProvider struct {
	db infraPostgres.Querier
}

func NewSubmissionStatsProvider(db infraPostgres.Querier) *SubmissionStatsProvider {
	return &SubmissionStatsProvider{db: db}
}

func (p *SubmissionStatsProvider) GetSubmissionCounts(ctx context.Context, userID string) (total int, accepted int, err error) {
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
