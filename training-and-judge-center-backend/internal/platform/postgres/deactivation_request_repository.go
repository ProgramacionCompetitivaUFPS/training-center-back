package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/training-judge-center/backend/internal/domain/user"
)

type DeactivationRequestRepository struct {
	pool *pgxpool.Pool
}

func NewDeactivationRequestRepository(pool *pgxpool.Pool) *DeactivationRequestRepository {
	return &DeactivationRequestRepository{pool: pool}
}

func (r *DeactivationRequestRepository) Save(ctx context.Context, req *user.DeactivationRequest) error {
	query := `
		INSERT INTO deactivation_requests 
		(id, user_id, verification_code, expires_at, attempts, blocked_until, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.pool.Exec(ctx, query,
		req.ID(),
		req.UserID(),
		req.VerificationCode(),
		req.ExpiresAt(),
		req.Attempts(),
		req.BlockedUntil(),
		string(req.Status()),
		req.CreatedAt(),
		req.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to save deactivation request: %w", err)
	}

	return nil
}

func (r *DeactivationRequestRepository) FindPendingByUserID(ctx context.Context, userID string) (*user.DeactivationRequest, error) {
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

	err := r.pool.QueryRow(ctx, query, userID, string(user.DeactivationStatusPending), string(user.DeactivationStatusBlocked)).Scan(
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
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find pending deactivation request: %w", err)
	}

	req := user.RestoreDeactivationRequest(id, returnedUserID, verificationCode, expiresAt, attempts, blockedUntil, user.DeactivationStatus(status), createdAt, updatedAt)
	return req, nil
}

func (r *DeactivationRequestRepository) FindByID(ctx context.Context, id string) (*user.DeactivationRequest, error) {
	query := `
		SELECT id, user_id, verification_code, expires_at, attempts, blocked_until, status, created_at, updated_at
		FROM deactivation_requests
		WHERE id = $1`

	var returnedID, returnedUserID, verificationCode string
	var expiresAt, createdAt, updatedAt time.Time
	var attempts int
	var blockedUntil *time.Time
	var status string

	err := r.pool.QueryRow(ctx, query, id).Scan(
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
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find deactivation request by id: %w", err)
	}

	req := user.RestoreDeactivationRequest(returnedID, returnedUserID, verificationCode, expiresAt, attempts, blockedUntil, user.DeactivationStatus(status), createdAt, updatedAt)
	return req, nil
}

func (r *DeactivationRequestRepository) Update(ctx context.Context, req *user.DeactivationRequest) error {
	query := `
		UPDATE deactivation_requests
		SET verification_code = $1, expires_at = $2, attempts = $3, blocked_until = $4, status = $5, updated_at = $6
		WHERE id = $7`

	_, err := r.pool.Exec(ctx, query,
		req.VerificationCode(),
		req.ExpiresAt(),
		req.Attempts(),
		req.BlockedUntil(),
		string(req.Status()),
		req.UpdatedAt(),
		req.ID(),
	)
	if err != nil {
		return fmt.Errorf("failed to update deactivation request: %w", err)
	}

	return nil
}

func (r *DeactivationRequestRepository) InvalidatePendingByUserID(ctx context.Context, userID string, now time.Time) error {
	query := `
		UPDATE deactivation_requests
		SET status = $1, updated_at = $2
		WHERE user_id = $3 AND status IN ($4, $5)`

	_, err := r.pool.Exec(ctx, query, 
		string(user.DeactivationStatusExpired), now, userID, string(user.DeactivationStatusPending), string(user.DeactivationStatusBlocked))
	
	if err != nil {
		return fmt.Errorf("failed to invalidate prior deactivation requests: %w", err)
	}

	return nil
}
