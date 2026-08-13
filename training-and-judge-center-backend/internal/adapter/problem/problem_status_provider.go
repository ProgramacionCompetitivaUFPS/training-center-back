package problem

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ProblemStatusProvider struct {
	db infraPostgres.Querier
}

func NewProblemStatusProvider(db infraPostgres.Querier) *ProblemStatusProvider {
	return &ProblemStatusProvider{db: db}
}

func (p *ProblemStatusProvider) GetStatus(ctx context.Context, problemID string) (string, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	var status string
	err := q.QueryRow(ctx, `SELECT status FROM problems WHERE id = $1`, problemID).Scan(&status)

	if errors.Is(err, pgx.ErrNoRows) {
		return "", apperror.NewNotFound(domainProblem.ErrCodeProblemNotFound, "problem not found")
	}
	if err != nil {
		slog.ErrorContext(ctx, "problem_status_provider: GetStatus failed", "error", err, "problem_id", problemID)
		return "", apperror.NewInternal()
	}
	return status, nil
}
