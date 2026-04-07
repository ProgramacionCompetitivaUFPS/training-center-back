package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/training-judge-center/backend/internal/domain/user"
)

type EmailChangeRepository struct {
	querier Querier
}

func NewEmailChangeRepository(querier Querier) *EmailChangeRepository {
	return &EmailChangeRepository{querier: querier}
}

func (r *EmailChangeRepository) Save(ctx context.Context, req *user.EmailChangeRequest) error {
	query := `
		INSERT INTO email_change_requests (id, user_id, new_email, code, status, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	querier := getQuerier(ctx, r.querier)
	_, err := querier.Exec(ctx, query,
		req.ID(),
		req.UserID(),
		req.NewEmail().String(),
		req.Code(),
		string(req.Status()),
		req.ExpiresAt(),
		req.CreatedAt(),
		req.UpdatedAt(),
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

	var returnedID, userID, code string
	var expiresAt, createdAt time.Time
	var updatedAt *time.Time
	var statusStr, emailStr string

	err := r.querier.QueryRow(ctx, query, id).Scan(
		&returnedID,
		&userID,
		&emailStr,
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
		return nil, fmt.Errorf("failed to find email change request: %w", err)
	}

	parsedEmail, err := user.NewEmail(emailStr)
	if err != nil {
		return nil, fmt.Errorf("invalid email stored in db for request %s: %w", returnedID, err)
	}
	req := user.RestoreEmailChangeRequest(returnedID, userID, parsedEmail, code, user.RequestStatus(statusStr), expiresAt, createdAt, updatedAt)

	return req, nil
}

func (r *EmailChangeRepository) Update(ctx context.Context, req *user.EmailChangeRequest) error {
	query := `
		UPDATE email_change_requests
		SET status = $1, updated_at = $2
		WHERE id = $3`

	querier := getQuerier(ctx, r.querier)
	_, err := querier.Exec(ctx, query,
		string(req.Status()),
		req.UpdatedAt(),
		req.ID(),
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

	var id, returnedUserID, codeStr string
	var expiresAt, createdAt time.Time
	var updatedAt *time.Time
	var statusStr, emailStr string

	err := r.querier.QueryRow(ctx, query, code, userID).Scan(
		&id,
		&returnedUserID,
		&emailStr,
		&codeStr,
		&statusStr,
		&expiresAt,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to find email change request by code: %w", err)
	}

	parsedEmail, err := user.NewEmail(emailStr)
	if err != nil {
		return nil, fmt.Errorf("invalid email stored in db for request %s: %w", id, err)
	}
	req := user.RestoreEmailChangeRequest(id, returnedUserID, parsedEmail, codeStr, user.RequestStatus(statusStr), expiresAt, createdAt, updatedAt)

	return req, nil
}

func (r *EmailChangeRepository) InvalidatePendingByUserID(ctx context.Context, userID string) error {
	query := `
		UPDATE email_change_requests
		SET status = $1, updated_at = $2
		WHERE user_id = $3 AND status = $4`

	_, err := r.querier.Exec(ctx, query,
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
