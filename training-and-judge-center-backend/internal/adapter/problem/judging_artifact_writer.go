package problem

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type JudgingArtifactWriter struct {
	db infraPostgres.Querier
}

func NewJudgingArtifactWriter(db infraPostgres.Querier) *JudgingArtifactWriter {
	return &JudgingArtifactWriter{db: db}
}

func (w *JudgingArtifactWriter) SetCheckerCompiledKey(ctx context.Context, problemID, compiledKey string, now time.Time) error {
	q := infraPostgres.GetQuerier(ctx, w.db)

	var raw []byte
	err := q.QueryRow(ctx, `SELECT checker FROM problems WHERE id = $1`, problemID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound(domainProblem.ErrCodeProblemNotFound, "problem not found")
	}
	if err != nil {
		slog.ErrorContext(ctx, "judging_artifact_writer: SetCheckerCompiledKey select failed", "error", err, "problem_id", problemID)
		return apperror.NewInternal()
	}
	if len(raw) == 0 {
		slog.ErrorContext(ctx, "judging_artifact_writer: no checker uploaded for problem", "problem_id", problemID)
		return apperror.NewInternal()
	}

	var file dbJudgingFile
	if err := json.Unmarshal(raw, &file); err != nil {
		slog.ErrorContext(ctx, "judging_artifact_writer: failed to parse checker JSON", "error", err, "problem_id", problemID)
		return apperror.NewInternal()
	}
	t := now.UTC()
	file.CompiledKey = &compiledKey
	file.CompiledAt = &t

	updated, err := json.Marshal(file)
	if err != nil {
		slog.ErrorContext(ctx, "judging_artifact_writer: failed to marshal updated checker JSON", "error", err, "problem_id", problemID)
		return apperror.NewInternal()
	}

	if _, err := q.Exec(ctx, `UPDATE problems SET checker = $2 WHERE id = $1`, problemID, updated); err != nil {
		slog.ErrorContext(ctx, "judging_artifact_writer: SetCheckerCompiledKey update failed", "error", err, "problem_id", problemID)
		return apperror.NewInternal()
	}
	return nil
}

func (w *JudgingArtifactWriter) SetValidatorCompiledKey(ctx context.Context, problemID, compiledKey string, now time.Time) error {
	q := infraPostgres.GetQuerier(ctx, w.db)

	var raw []byte
	err := q.QueryRow(ctx, `SELECT validator FROM problems WHERE id = $1`, problemID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound(domainProblem.ErrCodeProblemNotFound, "problem not found")
	}
	if err != nil {
		slog.ErrorContext(ctx, "judging_artifact_writer: SetValidatorCompiledKey select failed", "error", err, "problem_id", problemID)
		return apperror.NewInternal()
	}
	if len(raw) == 0 {
		slog.ErrorContext(ctx, "judging_artifact_writer: no validator uploaded for problem", "problem_id", problemID)
		return apperror.NewInternal()
	}

	var file dbJudgingFile
	if err := json.Unmarshal(raw, &file); err != nil {
		slog.ErrorContext(ctx, "judging_artifact_writer: failed to parse validator JSON", "error", err, "problem_id", problemID)
		return apperror.NewInternal()
	}
	t := now.UTC()
	file.CompiledKey = &compiledKey
	file.CompiledAt = &t

	updated, err := json.Marshal(file)
	if err != nil {
		slog.ErrorContext(ctx, "judging_artifact_writer: failed to marshal updated validator JSON", "error", err, "problem_id", problemID)
		return apperror.NewInternal()
	}

	if _, err := q.Exec(ctx, `UPDATE problems SET validator = $2 WHERE id = $1`, problemID, updated); err != nil {
		slog.ErrorContext(ctx, "judging_artifact_writer: SetValidatorCompiledKey update failed", "error", err, "problem_id", problemID)
		return apperror.NewInternal()
	}
	return nil
}
