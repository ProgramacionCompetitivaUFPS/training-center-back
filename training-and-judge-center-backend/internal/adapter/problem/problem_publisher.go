package problem

import (
	"context"
	"log/slog"
	"time"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ProblemPublisher struct {
	db infraPostgres.Querier
}

func NewProblemPublisher(db infraPostgres.Querier) *ProblemPublisher {
	return &ProblemPublisher{db: db}
}

func (p *ProblemPublisher) MarkPublished(ctx context.Context, problemID string, now time.Time) error {
	q := infraPostgres.GetQuerier(ctx, p.db)

	tag, err := q.Exec(ctx, `UPDATE problems SET status = 'PUBLISHED', updated_at = $2 WHERE id = $1`, problemID, now)
	if err != nil {
		slog.ErrorContext(ctx, "problem_publisher: MarkPublished failed", "error", err, "problem_id", problemID)
		return apperror.NewInternal()
	}
	if tag.RowsAffected() == 0 {
		return apperror.NewNotFound(domainProblem.ErrCodeProblemNotFound, "problem not found")
	}
	return nil
}
