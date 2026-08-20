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

type ProblemProvider struct {
	db infraPostgres.Querier
}

func NewProblemProvider(db infraPostgres.Querier) *ProblemProvider {
	return &ProblemProvider{db: db}
}

// The stored JSON also carries the uploaded filename, which the judge no longer
// needs: the artifact has a fixed name inside the sandbox.
type dbCheckerJSON struct {
	FileKey     string  `json:"fileKey"`
	Language    string  `json:"language"`
	CompiledKey *string `json:"compiledKey,omitempty"`
}

func (p *ProblemProvider) GetLimits(ctx context.Context, problemID string) (appJudge.ProblemLimits, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	var timeLimitMs, memoryMb *int
	var checkerJSON []byte

	err := q.QueryRow(ctx,
		`SELECT time_limit, memory_limit, checker FROM problems WHERE id = $1`,
		problemID,
	).Scan(&timeLimitMs, &memoryMb, &checkerJSON)

	if errors.Is(err, pgx.ErrNoRows) {
		return appJudge.ProblemLimits{}, apperror.NewNotFound(apperror.ErrCodeNotFound, "problem not found")
	}
	if err != nil {
		slog.ErrorContext(ctx, "problem_provider: GetLimits query failed", "error", err, "problem_id", problemID)
		return appJudge.ProblemLimits{}, apperror.NewInternal()
	}

	limits := appJudge.ProblemLimits{}
	if timeLimitMs != nil {
		limits.TimeLimitMs = *timeLimitMs
	}
	if memoryMb != nil {
		limits.MemoryKb = *memoryMb * 1024
	}

	if len(checkerJSON) > 0 {
		var checker dbCheckerJSON
		if err := json.Unmarshal(checkerJSON, &checker); err != nil {
			slog.ErrorContext(ctx, "problem_provider: failed to parse checker JSON", "error", err, "problem_id", problemID)
			return appJudge.ProblemLimits{}, apperror.NewInternal()
		}
		limits.HasCustomChecker = true

		if checker.CompiledKey == nil {
			// A checker that was never compiled: publish validation should
			// have blocked this problem from ever reaching PUBLISHED, so
			// this is an anomaly, not a normal state — worth surfacing, but
			// falling back to token comparison (CheckerPath left empty) is
			// safer than failing every judge run for this problem.
			slog.WarnContext(ctx, "problem_provider: checker has no compiled artifact", "problem_id", problemID)
		} else {
			lang, err := submission.NewLanguage(checker.Language)
			if err != nil {
				slog.ErrorContext(ctx, "problem_provider: invalid checker language", "error", err, "problem_id", problemID, "language", checker.Language)
				return appJudge.ProblemLimits{}, apperror.NewInternal()
			}
			limits.CheckerPath = *checker.CompiledKey
			limits.CheckerLanguage = lang
		}
	}

	return limits, nil
}
