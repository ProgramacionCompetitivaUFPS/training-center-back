package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/training-judge-center/backend/internal/domain/user"
)

type EmailChangeRepository struct {
	pool *pgxpool.Pool
}

func NewEmailChangeRepository(pool *pgxpool.Pool) *EmailChangeRepository {
	return &EmailChangeRepository{pool: pool}
}

func (r *EmailChangeRepository) Save(ctx context.Context, req *user.EmailChangeRequest) error {
	query := `
		INSERT INTO email_change_requests (id, user_id, new_email, code, status, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.pool.Exec(ctx, query,
		req.ID,
		req.UserID,
		req.NewEmail.String(),
		req.Code,
		string(req.Status),
		req.ExpiresAt,
		req.CreatedAt,
		req.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save email change request: %w", err)
	}
	return nil
}

func (r *EmailChangeRepository) FindByID(ctx context.Context, id string) (*user.EmailChangeRequest, error) {
	query := `
		SELECT id, user_id, new_email, code, status, expires_at, created_at, updated_at
		FROM email_change_requests
		WHERE id = $1`

	var req user.EmailChangeRequest
	var statusStr, emailStr string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&req.ID,
		&req.UserID,
		&emailStr,
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
		return nil, fmt.Errorf("failed to find email change request: %w", err)
	}

	req.Status = user.RequestStatus(statusStr)
	parsedEmail, _ := user.NewEmail(emailStr)
	req.NewEmail = parsedEmail

	return &req, nil
}

func (r *EmailChangeRepository) Update(ctx context.Context, req *user.EmailChangeRequest) error {
	query := `
		UPDATE email_change_requests
		SET status = $1, updated_at = $2
		WHERE id = $3`

	_, err := r.pool.Exec(ctx, query,
		string(req.Status),
		req.UpdatedAt,
		req.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update email change request: %w", err)
	}
	return nil
}

func (r *EmailChangeRepository) FindByCodeAndUserID(ctx context.Context, code string, userID string) (*user.EmailChangeRequest, error) {
	query := `
		SELECT id, user_id, new_email, code, status, expires_at, created_at, updated_at
		FROM email_change_requests
		WHERE code = $1 AND user_id = $2`

	var req user.EmailChangeRequest
	var statusStr, emailStr string
	err := r.pool.QueryRow(ctx, query, code, userID).Scan(
		&req.ID,
		&req.UserID,
		&emailStr,
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
		return nil, fmt.Errorf("failed to find email change request by code: %w", err)
	}

	req.Status = user.RequestStatus(statusStr)
	parsedEmail, _ := user.NewEmail(emailStr)
	req.NewEmail = parsedEmail

	return &req, nil
}

func (r *EmailChangeRepository) InvalidatePendingByUserID(ctx context.Context, userID string) error {
	query := `
		UPDATE email_change_requests
		SET status = $1, updated_at = $2
		WHERE user_id = $3 AND status = $4`

	_, err := r.pool.Exec(ctx, query,
		string(user.StatusExpired),
		time.Now(),
		userID,
		string(user.StatusPending),
	)
	if err != nil {
		return fmt.Errorf("failed to invalidate pending email change requests: %w", err)
	}
	return nil
}
