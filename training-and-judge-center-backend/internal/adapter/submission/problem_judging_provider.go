package submission

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ProblemJudgingProvider implements appsubmission.ProblemJudgingProvider.
type ProblemJudgingProvider struct {
	db infraPostgres.Querier
}

func NewProblemJudgingProvider(db infraPostgres.Querier) *ProblemJudgingProvider {
	return &ProblemJudgingProvider{db: db}
}

func (p *ProblemJudgingProvider) GetJudgingUpdatedAt(ctx context.Context, problemID string) (*time.Time, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)
	var t *time.Time
	err := q.QueryRow(ctx,
		`SELECT judging_updated_at FROM problems WHERE id = $1`, problemID,
	).Scan(&t)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "submission: failed to get judging_updated_at", "problem_id", problemID, "error", err)
		return nil, apperror.NewInternal()
	}
	return t, nil
}

func (p *ProblemJudgingProvider) IsPublished(ctx context.Context, problemID string) (bool, error) {
	if problemID == "" {
		return false, nil
	}
	q := infraPostgres.GetQuerier(ctx, p.db)
	var published bool
	err := q.QueryRow(ctx,
		`SELECT status = 'PUBLISHED' FROM problems WHERE id = $1`, problemID,
	).Scan(&published)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		slog.ErrorContext(ctx, "submission: failed to get problem status", "problem_id", problemID, "error", err)
		return false, apperror.NewInternal()
	}
	return published, nil
}
