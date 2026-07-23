package user

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	domainUser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type DeactivationRequestRepository struct {
	querier infraPostgres.Querier
}

func NewDeactivationRequestRepository(querier infraPostgres.Querier) *DeactivationRequestRepository {
	return &DeactivationRequestRepository{querier: querier}
}

func (r *DeactivationRequestRepository) Save(ctx context.Context, req *domainUser.DeactivationRequest) error {
	query := `
		INSERT INTO deactivation_requests
		(id, user_id, verification_code, expires_at, attempts, blocked_until, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	querier := infraPostgres.GetQuerier(ctx, r.querier)
	_, err := querier.Exec(ctx, query,
		req.ID(),
		req.UserID(),
		req.VerificationCode(),
		req.ExpiresAt(),
		req.Attempts(),
		req.BlockedUntil(),
		req.Status().String(),
		req.CreatedAt(),
		req.UpdatedAt(),
	)
	if err != nil {
		slog.ErrorContext(ctx, "database error saving deactivation request", "error", err)
		return apperror.NewInternal()
	}

	return nil
}

func (r *DeactivationRequestRepository) FindPendingByUserID(ctx context.Context, userID string) (*domainUser.DeactivationRequest, error) {
	query := `
		SELECT id, user_id, verification_code, expires_at, attempts, blocked_until, status, created_at, updated_at
		FROM deactivation_requests
		WHERE user_id = $1 AND status IN ($2, $3)
		ORDER BY created_at DESC LIMIT 1`

	var id, returnedUserID, verificationCode string
	var expiresAt, createdAt, updatedAt time.Time
	var attempts int
	var blockedUntil *time.Time
	var status string

	err := r.querier.QueryRow(ctx, query, userID, domainUser.DeactivationStatusPending.String(), domainUser.DeactivationStatusBlocked.String()).Scan(
		&id,
		&returnedUserID,
		&verificationCode,
		&expiresAt,
		&attempts,
		&blockedUntil,
		&status,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "database error finding pending deactivation request", "user_id", userID, "error", err)
		return nil, apperror.NewInternal()
	}

	req := domainUser.RestoreDeactivationRequest(id, returnedUserID, verificationCode, expiresAt, attempts, blockedUntil, domainUser.RestoreDeactivationStatus(status), createdAt, updatedAt)
	return req, nil
}

func (r *DeactivationRequestRepository) FindByID(ctx context.Context, id string) (*domainUser.DeactivationRequest, error) {
	query := `
		SELECT id, user_id, verification_code, expires_at, attempts, blocked_until, status, created_at, updated_at
		FROM deactivation_requests
		WHERE id = $1`

	var returnedID, returnedUserID, verificationCode string
	var expiresAt, createdAt, updatedAt time.Time
	var attempts int
	var blockedUntil *time.Time
	var status string

	err := r.querier.QueryRow(ctx, query, id).Scan(
		&returnedID,
		&returnedUserID,
		&verificationCode,
		&expiresAt,
		&attempts,
		&blockedUntil,
		&status,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "database error finding deactivation request by id", "id", id, "error", err)
		return nil, apperror.NewInternal()
	}

	req := domainUser.RestoreDeactivationRequest(returnedID, returnedUserID, verificationCode, expiresAt, attempts, blockedUntil, domainUser.RestoreDeactivationStatus(status), createdAt, updatedAt)
	return req, nil
}

func (r *DeactivationRequestRepository) Update(ctx context.Context, req *domainUser.DeactivationRequest) error {
	query := `
		UPDATE deactivation_requests
		SET verification_code = $1, expires_at = $2, attempts = $3, blocked_until = $4, status = $5, updated_at = $6
		WHERE id = $7`

	querier := infraPostgres.GetQuerier(ctx, r.querier)
	_, err := querier.Exec(ctx, query,
		req.VerificationCode(),
		req.ExpiresAt(),
		req.Attempts(),
		req.BlockedUntil(),
		req.Status().String(),
		req.UpdatedAt(),
		req.ID(),
	)
	if err != nil {
		slog.ErrorContext(ctx, "database error updating deactivation request", "id", req.ID(), "error", err)
		return apperror.NewInternal()
	}

	return nil
}

func (r *DeactivationRequestRepository) InvalidatePendingByUserID(ctx context.Context, userID string, now time.Time) error {
	query := `
		UPDATE deactivation_requests
		SET status = $1, updated_at = $2
		WHERE user_id = $3 AND status IN ($4, $5)`

	querier := infraPostgres.GetQuerier(ctx, r.querier)
	_, err := querier.Exec(ctx, query,
		domainUser.DeactivationStatusExpired.String(), now, userID, domainUser.DeactivationStatusPending.String(), domainUser.DeactivationStatusBlocked.String())

	if err != nil {
		slog.ErrorContext(ctx, "database error invalidating deactivation requests", "user_id", userID, "error", err)
		return apperror.NewInternal()
	}

	return nil
}
