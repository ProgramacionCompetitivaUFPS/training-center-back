package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/training-judge-center/backend/internal/domain/user"
)

type PasswordRecoveryRepository struct {
	pool *pgxpool.Pool
}

func NewPasswordRecoveryRepository(pool *pgxpool.Pool) *PasswordRecoveryRepository {
	return &PasswordRecoveryRepository{pool: pool}
}

func (r *PasswordRecoveryRepository) Save(ctx context.Context, req *user.PasswordRecoveryRequest) error {
	query := `
		INSERT INTO password_recovery_requests (id, user_id, code, status, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.pool.Exec(ctx, query,
		req.ID,
		req.UserID,
		req.Code,
		string(req.Status),
		req.ExpiresAt,
		req.CreatedAt,
		req.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save password recovery request: %w", err)
	}
	return nil
}

func (r *PasswordRecoveryRepository) FindByID(ctx context.Context, id string) (*user.PasswordRecoveryRequest, error) {
	query := `
		SELECT id, user_id, code, status, expires_at, created_at, updated_at
		FROM password_recovery_requests
		WHERE id = $1`

	var req user.PasswordRecoveryRequest
	var statusStr string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&req.ID,
		&req.UserID,
		&req.Code,
		&statusStr,
		&req.ExpiresAt,
		&req.CreatedAt,
		&req.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to find password recovery request: %w", err)
	}

	req.Status = user.RequestStatus(statusStr)
	return &req, nil
}

func (r *PasswordRecoveryRepository) Update(ctx context.Context, req *user.PasswordRecoveryRequest) error {
	query := `
		UPDATE password_recovery_requests
		SET status = $1, updated_at = $2
		WHERE id = $3`

	_, err := r.pool.Exec(ctx, query,
		string(req.Status),
		req.UpdatedAt,
		req.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update password recovery request: %w", err)
	}
	return nil
}

func (r *PasswordRecoveryRepository) FindPendingByUserID(ctx context.Context, userID string) (*user.PasswordRecoveryRequest, error) {
	query := `
		SELECT id, user_id, code, status, expires_at, created_at, updated_at
		FROM password_recovery_requests
		WHERE user_id = $1 AND status = $2`

	var req user.PasswordRecoveryRequest
	var statusStr string
	err := r.pool.QueryRow(ctx, query, userID, string(user.StatusPending)).Scan(
		&req.ID,
		&req.UserID,
		&req.Code,
		&statusStr,
		&req.ExpiresAt,
		&req.CreatedAt,
		&req.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to find pending password recovery request: %w", err)
	}

	req.Status = user.RequestStatus(statusStr)
	return &req, nil
}

func (r *PasswordRecoveryRepository) InvalidatePendingByUserID(ctx context.Context, userID string, now time.Time) error {
	query := `
		UPDATE password_recovery_requests
		SET status = $1, updated_at = $2
		WHERE user_id = $3 AND status = $4`

	_, err := r.pool.Exec(ctx, query, string(user.StatusExpired), now, userID, string(user.StatusPending))
	if err != nil {
		return fmt.Errorf("failed to invalidate pending password recovery requests: %w", err)
	}
	return nil
}
