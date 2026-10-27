package problem

import (
	"context"
	"log/slog"
	"time"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var _ appProblem.StaleValidationRecoverer = (*StaleValidationRecoverer)(nil)

type StaleValidationRecoverer struct {
	db infraPostgres.Querier
}

func NewStaleValidationRecoverer(db infraPostgres.Querier) *StaleValidationRecoverer {
	return &StaleValidationRecoverer{db: db}
}

// RecoverStaleBefore marks RUNNING validations still requested before cutoff
// as SYSTEM_ERROR. There's no updated_at on problem_validations — Start()
// doesn't bump one — so requested_at is the only timestamp available, and
// it's a close enough proxy: ValidateProblemUseCase marks a ticket RUNNING
// right after loading it, moments after it was requested.
func (r *StaleValidationRecoverer) RecoverStaleBefore(ctx context.Context, cutoff time.Time) (int, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)
	tag, err := q.Exec(ctx, `
		UPDATE problem_validations SET
			status       = 'SYSTEM_ERROR',
			completed_at = now()
		WHERE status = 'RUNNING' AND requested_at < $1
	`, cutoff)
	if err != nil {
		slog.ErrorContext(ctx, "problem: failed to recover stale validations", "error", err)
		return 0, apperror.NewInternal()
	}
	return int(tag.RowsAffected()), nil
}
