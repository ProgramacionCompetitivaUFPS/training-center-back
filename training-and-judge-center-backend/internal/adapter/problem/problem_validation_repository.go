package problem

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ProblemValidationRepository struct {
	db infraPostgres.Querier
}

func NewProblemValidationRepository(db infraPostgres.Querier) *ProblemValidationRepository {
	return &ProblemValidationRepository{db: db}
}

func (r *ProblemValidationRepository) Save(ctx context.Context, v *domainProblem.ProblemValidation) error {
	q := infraPostgres.GetQuerier(ctx, r.db)

	_, err := q.Exec(ctx,
		`INSERT INTO problem_validations (id, problem_id, requested_by, status, requested_at, completed_at, result)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (id) DO UPDATE SET
		   status = EXCLUDED.status,
		   completed_at = EXCLUDED.completed_at,
		   result = EXCLUDED.result`,
		v.ID(), v.ProblemID(), v.RequestedBy().String(), v.Status().String(), v.RequestedAt(), v.CompletedAt(), v.ResultJSON(),
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == infraPostgres.UniqueViolation && pgErr.ConstraintName == "idx_problem_validations_active_per_problem" {
			return apperror.NewConflict(domainProblem.ErrCodeValidationInProgress, "a validation is already in progress for this problem")
		}
		slog.ErrorContext(ctx, "problem_validation_repository: Save failed", "error", err, "validation_id", v.ID())
		return apperror.NewInternal()
	}
	return nil
}

func (r *ProblemValidationRepository) FindByID(ctx context.Context, id string) (*domainProblem.ProblemValidation, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)

	var problemID, requestedBy, status string
	var requestedAt time.Time
	var completedAt *time.Time
	var result *string

	err := q.QueryRow(ctx,
		`SELECT problem_id, requested_by, status, requested_at, completed_at, result
		 FROM problem_validations WHERE id = $1`,
		id,
	).Scan(&problemID, &requestedBy, &status, &requestedAt, &completedAt, &result)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "problem validation not found")
	}
	if err != nil {
		slog.ErrorContext(ctx, "problem_validation_repository: FindByID failed", "error", err, "validation_id", id)
		return nil, apperror.NewInternal()
	}

	return domainProblem.RestoreProblemValidation(
		id, problemID, shared.RestoreUserID(requestedBy),
		domainProblem.RestoreProblemValidationStatus(status),
		requestedAt, completedAt, result,
	), nil
}

func (r *ProblemValidationRepository) FindLatestByProblemID(ctx context.Context, problemID string) (*domainProblem.ProblemValidation, bool, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)

	var id, requestedBy, status string
	var requestedAt time.Time
	var completedAt *time.Time
	var result *string

	err := q.QueryRow(ctx,
		`SELECT id, requested_by, status, requested_at, completed_at, result
		 FROM problem_validations WHERE problem_id = $1
		 ORDER BY requested_at DESC LIMIT 1`,
		problemID,
	).Scan(&id, &requestedBy, &status, &requestedAt, &completedAt, &result)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		slog.ErrorContext(ctx, "problem_validation_repository: FindLatestByProblemID failed", "error", err, "problem_id", problemID)
		return nil, false, apperror.NewInternal()
	}

	v := domainProblem.RestoreProblemValidation(
		id, problemID, shared.RestoreUserID(requestedBy),
		domainProblem.RestoreProblemValidationStatus(status),
		requestedAt, completedAt, result,
	)
	return v, true, nil
}
