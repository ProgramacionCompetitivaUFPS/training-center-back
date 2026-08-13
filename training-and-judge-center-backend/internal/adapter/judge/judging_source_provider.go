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

type JudgingSourceProvider struct {
	db infraPostgres.Querier
}

func NewJudgingSourceProvider(db infraPostgres.Querier) *JudgingSourceProvider {
	return &JudgingSourceProvider{db: db}
}

type dbJudgingSourceFile struct {
	Filename string `json:"filename"`
	FileKey  string `json:"fileKey"`
	Language string `json:"language"`
}

func (p *JudgingSourceProvider) GetCheckerSource(ctx context.Context, problemID string) (*appJudge.JudgingSource, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	var raw []byte
	err := q.QueryRow(ctx, `SELECT checker FROM problems WHERE id = $1`, problemID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "problem not found")
	}
	if err != nil {
		slog.ErrorContext(ctx, "judging_source_provider: GetCheckerSource query failed", "error", err, "problem_id", problemID)
		return nil, apperror.NewInternal()
	}
	if len(raw) == 0 {
		return nil, nil
	}

	var file dbJudgingSourceFile
	if err := json.Unmarshal(raw, &file); err != nil {
		slog.ErrorContext(ctx, "judging_source_provider: failed to parse checker JSON", "error", err, "problem_id", problemID)
		return nil, apperror.NewInternal()
	}
	lang, err := submission.NewLanguage(file.Language)
	if err != nil {
		slog.ErrorContext(ctx, "judging_source_provider: invalid checker language", "error", err, "problem_id", problemID, "language", file.Language)
		return nil, apperror.NewInternal()
	}
	return &appJudge.JudgingSource{Filename: file.Filename, FileKey: file.FileKey, Language: lang}, nil
}

func (p *JudgingSourceProvider) GetValidatorSource(ctx context.Context, problemID string) (*appJudge.JudgingSource, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	var raw []byte
	err := q.QueryRow(ctx, `SELECT validator FROM problems WHERE id = $1`, problemID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "problem not found")
	}
	if err != nil {
		slog.ErrorContext(ctx, "judging_source_provider: GetValidatorSource query failed", "error", err, "problem_id", problemID)
		return nil, apperror.NewInternal()
	}
	if len(raw) == 0 {
		return nil, nil
	}

	var file dbJudgingSourceFile
	if err := json.Unmarshal(raw, &file); err != nil {
		slog.ErrorContext(ctx, "judging_source_provider: failed to parse validator JSON", "error", err, "problem_id", problemID)
		return nil, apperror.NewInternal()
	}
	lang, err := submission.NewLanguage(file.Language)
	if err != nil {
		slog.ErrorContext(ctx, "judging_source_provider: invalid validator language", "error", err, "problem_id", problemID, "language", file.Language)
		return nil, apperror.NewInternal()
	}
	return &appJudge.JudgingSource{Filename: file.Filename, FileKey: file.FileKey, Language: lang}, nil
}
