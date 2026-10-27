package judge

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appJudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type SolutionProvider struct {
	db infraPostgres.Querier
}

func NewSolutionProvider(db infraPostgres.Querier) *SolutionProvider {
	return &SolutionProvider{db: db}
}

type dbSolutionFile struct {
	FileKey  string `json:"fileKey"`
	Language string `json:"language"`
}

func (p *SolutionProvider) GetSolutions(ctx context.Context, problemID string) ([]appJudge.Solution, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	var solutionsJSON []byte
	err := q.QueryRow(ctx, `SELECT solutions FROM problems WHERE id = $1`, problemID).Scan(&solutionsJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "problem not found")
	}
	if err != nil {
		slog.ErrorContext(ctx, "solution_provider: query failed", "error", err, "problem_id", problemID)
		return nil, apperror.NewInternal()
	}

	if len(solutionsJSON) == 0 {
		return []appJudge.Solution{}, nil
	}

	var dbSolutions []dbSolutionFile
	if err := json.Unmarshal(solutionsJSON, &dbSolutions); err != nil {
		slog.ErrorContext(ctx, "solution_provider: failed to parse solutions JSON", "error", err, "problem_id", problemID)
		return nil, apperror.NewInternal()
	}

	solutions := make([]appJudge.Solution, 0, len(dbSolutions))
	for _, s := range dbSolutions {
		lang, err := submission.NewLanguage(s.Language)
		if err != nil {
			slog.ErrorContext(ctx, "solution_provider: invalid language in stored solution", "error", err, "problem_id", problemID, "language", s.Language)
			return nil, apperror.NewInternal()
		}
		solutions = append(solutions, appJudge.Solution{FileKey: s.FileKey, Language: lang})
	}
	return solutions, nil
}
