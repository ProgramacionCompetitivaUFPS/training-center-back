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

func (p *ProblemDisplayProvider) GetProblemsByIDs(ctx context.Context, problemIDs []string) (map[string]*appsubmission.ProblemDisplay, error) {
	if len(problemIDs) == 0 {
		return map[string]*appsubmission.ProblemDisplay{}, nil
	}
	q := infraPostgres.GetQuerier(ctx, p.db)

	rows, err := q.Query(ctx, `SELECT id, slug, title FROM problems WHERE id = ANY($1)`, problemIDs)
	if err != nil {
		slog.ErrorContext(ctx, "submission: failed to get problem displays", "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	result := make(map[string]*appsubmission.ProblemDisplay, len(problemIDs))
	for rows.Next() {
		var id, slug, title string
		if err := rows.Scan(&id, &slug, &title); err != nil {
			slog.ErrorContext(ctx, "submission: failed to scan problem display row", "error", err)
			return nil, apperror.NewInternal()
		}
		result[id] = &appsubmission.ProblemDisplay{Slug: slug, Title: title}
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "submission: error iterating problem display rows", "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}
