package user

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	domainUser "github.com/training-judge-center/backend/internal/domain/user"
	infraPostgres "github.com/training-judge-center/backend/internal/infrastructure/postgres"
)

type EmailChangeRepository struct {
	querier infraPostgres.Querier
}

func NewEmailChangeRepository(querier infraPostgres.Querier) *EmailChangeRepository {
	return &EmailChangeRepository{querier: querier}
}

func (r *EmailChangeRepository) Save(ctx context.Context, req *domainUser.EmailChangeRequest) error {
	query := `
		INSERT INTO email_change_requests (id, user_id, new_email, code, status, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	querier := infraPostgres.GetQuerier(ctx, r.querier)
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

func (r *EmailChangeRepository) FindByID(ctx context.Context, id string) (*domainUser.EmailChangeRequest, error) {
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

	parsedEmail, err := domainUser.NewEmail(emailStr)
	if err != nil {
		return nil, fmt.Errorf("invalid email stored in db for request %s: %w", returnedID, err)
	}
	status, err := domainUser.NewRequestStatus(statusStr)
	if err != nil {
		return nil, fmt.Errorf("invalid status in email change request %s: %w", returnedID, err)
	}
	req := domainUser.RestoreEmailChangeRequest(returnedID, userID, parsedEmail, code, status, expiresAt, createdAt, updatedAt)

	return req, nil
}

func (r *EmailChangeRepository) Update(ctx context.Context, req *domainUser.EmailChangeRequest) error {
	query := `
		UPDATE email_change_requests
		SET status = $1, updated_at = $2
		WHERE id = $3`

	querier := infraPostgres.GetQuerier(ctx, r.querier)
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

func (r *EmailChangeRepository) FindByCodeAndUserID(ctx context.Context, code string, userID string) (*domainUser.EmailChangeRequest, error) {
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

	parsedEmail, err := domainUser.NewEmail(emailStr)
	if err != nil {
		return nil, fmt.Errorf("invalid email stored in db for request %s: %w", id, err)
	}
	status, err := domainUser.NewRequestStatus(statusStr)
	if err != nil {
		return nil, fmt.Errorf("invalid status in email change request %s: %w", id, err)
	}
	req := domainUser.RestoreEmailChangeRequest(id, returnedUserID, parsedEmail, codeStr, status, expiresAt, createdAt, updatedAt)

	return req, nil
}

func (r *EmailChangeRepository) InvalidatePendingByUserID(ctx context.Context, userID string) error {
	query := `
		UPDATE email_change_requests
		SET status = $1, updated_at = $2
		WHERE user_id = $3 AND status = $4`

	querier := infraPostgres.GetQuerier(ctx, r.querier)
	_, err := querier.Exec(ctx, query,
		string(domainUser.StatusExpired),
		time.Now(),
		userID,
		string(domainUser.StatusPending),
	)
	if err != nil {
		return fmt.Errorf("failed to invalidate pending email change requests: %w", err)
	}
	return nil
}
