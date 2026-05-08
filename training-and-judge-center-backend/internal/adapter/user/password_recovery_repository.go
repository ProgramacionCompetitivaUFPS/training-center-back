package user

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	domainUser "github.com/training-judge-center/backend/internal/domain/user"
)

type PasswordRecoveryRepository struct {
	querier infraPostgres.Querier
}

func NewPasswordRecoveryRepository(querier infraPostgres.Querier) *PasswordRecoveryRepository {
	return &PasswordRecoveryRepository{querier: querier}
}

func (r *PasswordRecoveryRepository) Save(ctx context.Context, req *domainUser.PasswordRecoveryRequest) error {
	query := `
		INSERT INTO password_recovery_requests (id, user_id, code, status, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	querier := infraPostgres.GetQuerier(ctx, r.querier)
	_, err := querier.Exec(ctx, query,
		req.ID(),
		req.UserID(),
		req.Code(),
		req.Status().String(),
		req.ExpiresAt(),
		req.CreatedAt(),
		req.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to save password recovery request: %w", err)
	}
	return nil
}

func (r *PasswordRecoveryRepository) FindByID(ctx context.Context, id string) (*domainUser.PasswordRecoveryRequest, error) {
	query := `
		SELECT id, user_id, code, status, expires_at, created_at, updated_at
		FROM password_recovery_requests
		WHERE id = $1`

	var returnedID, userID, code string
	var expiresAt, createdAt time.Time
	var updatedAt *time.Time
	var statusStr string

	err := r.querier.QueryRow(ctx, query, id).Scan(
		&returnedID,
		&userID,
		&code,
		&statusStr,
		&expiresAt,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to find password recovery request: %w", err)
	}

	status, err := domainUser.NewRequestStatus(statusStr)
	if err != nil {
		return nil, fmt.Errorf("invalid status in password recovery request: %w", err)
	}
	req := domainUser.RestorePasswordRecoveryRequest(returnedID, userID, code, status, expiresAt, createdAt, updatedAt)
	return req, nil
}

func (r *PasswordRecoveryRepository) Update(ctx context.Context, req *domainUser.PasswordRecoveryRequest) error {
	query := `
		UPDATE password_recovery_requests
		SET status = $1, updated_at = $2
		WHERE id = $3`

	querier := infraPostgres.GetQuerier(ctx, r.querier)
	_, err := querier.Exec(ctx, query,
		req.Status().String(),
		req.UpdatedAt(),
		req.ID(),
	)
	if err != nil {
		return fmt.Errorf("failed to update password recovery request: %w", err)
	}
	return nil
}

func (r *PasswordRecoveryRepository) FindPendingByUserID(ctx context.Context, userID string) (*domainUser.PasswordRecoveryRequest, error) {
	query := `
		SELECT id, user_id, code, status, expires_at, created_at, updated_at
		FROM password_recovery_requests
		WHERE user_id = $1 AND status = $2`

	var id, returnedUserID, code string
	var expiresAt, createdAt time.Time
	var updatedAt *time.Time
	var statusStr string

	err := r.querier.QueryRow(ctx, query, userID, domainUser.RequestStatusPending.String()).Scan(
		&id,
		&returnedUserID,
		&code,
		&statusStr,
		&expiresAt,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to find pending password recovery request: %w", err)
	}

	status, err := domainUser.NewRequestStatus(statusStr)
	if err != nil {
		return nil, fmt.Errorf("invalid status in pending password recovery request: %w", err)
	}
	req := domainUser.RestorePasswordRecoveryRequest(id, returnedUserID, code, status, expiresAt, createdAt, updatedAt)
	return req, nil
}

func (r *PasswordRecoveryRepository) InvalidatePendingByUserID(ctx context.Context, userID string, now time.Time) error {
	query := `
		UPDATE password_recovery_requests
		SET status = $1, updated_at = $2
		WHERE user_id = $3 AND status = $4`

	querier := infraPostgres.GetQuerier(ctx, r.querier)
	_, err := querier.Exec(ctx, query, domainUser.RequestStatusExpired.String(), now, userID, domainUser.RequestStatusPending.String())
	if err != nil {
		return fmt.Errorf("failed to invalidate pending password recovery requests: %w", err)
	}
	return nil
}
