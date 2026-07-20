package judge

import (
	"context"
	"log/slog"
	"time"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appJudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var _ appJudge.StaleSubmissionRecoverer = (*StaleSubmissionRecoverer)(nil)

type StaleSubmissionRecoverer struct {
	db infraPostgres.Querier
}

func NewStaleSubmissionRecoverer(db infraPostgres.Querier) *StaleSubmissionRecoverer {
	return &StaleSubmissionRecoverer{db: db}
}

func (r *StaleSubmissionRecoverer) RecoverStaleBefore(ctx context.Context, cutoff time.Time) (int, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)
	tag, err := q.Exec(ctx, `
		UPDATE submissions SET
			status     = 'SYSTEM_ERROR',
			updated_at = now()
		WHERE status = 'RUNNING' AND updated_at < $1
	`, cutoff)
	if err != nil {
		slog.ErrorContext(ctx, "judge: failed to recover stale submissions", "error", err)
		return 0, apperror.NewInternal()
	}
	return int(tag.RowsAffected()), nil
}
