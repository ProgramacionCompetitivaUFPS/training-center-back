package submission

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
	domainproblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ProblemDisplayProvider struct {
	db infraPostgres.Querier
}

func NewProblemDisplayProvider(db infraPostgres.Querier) *ProblemDisplayProvider {
	return &ProblemDisplayProvider{db: db}
}

func (p *ProblemDisplayProvider) GetProblemByID(ctx context.Context, problemID string) (*appsubmission.ProblemDisplay, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	var slug, title string
	err := q.QueryRow(ctx, `SELECT slug, title FROM problems WHERE id = $1`, problemID).Scan(&slug, &title)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound(domainproblem.ErrCodeProblemNotFound, "problem not found")
		}
		slog.ErrorContext(ctx, "submission: failed to get problem display", "problem_id", problemID, "error", err)
		return nil, apperror.NewInternal()
	}

	return &appsubmission.ProblemDisplay{Slug: slug, Title: title}, nil
}
