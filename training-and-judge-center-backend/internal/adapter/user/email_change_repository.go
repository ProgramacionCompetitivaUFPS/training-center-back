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
		req.Status().String(),
		req.ExpiresAt(),
		req.CreatedAt(),
		req.UpdatedAt(),
	)
	if err != nil {
		slog.ErrorContext(ctx, "database error saving email change request", "error", err)
		return apperror.NewInternal()
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "database error finding email change request by id", "id", id, "error", err)
		return nil, apperror.NewInternal()
	}

	parsedEmail := domainUser.RestoreEmail(emailStr)
	status, err := domainUser.NewRequestStatus(statusStr)
	if err != nil {
		slog.ErrorContext(ctx, "corrupted status in email_change_requests table", "id", returnedID, "status", statusStr, "error", err)
		return nil, apperror.NewInternal()
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
		req.Status().String(),
		req.UpdatedAt(),
		req.ID(),
	)
	if err != nil {
		slog.ErrorContext(ctx, "database error updating email change request", "id", req.ID(), "error", err)
		return apperror.NewInternal()
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "database error finding email change request by code", "user_id", userID, "error", err)
		return nil, apperror.NewInternal()
	}

	parsedEmail := domainUser.RestoreEmail(emailStr)
	status, err := domainUser.NewRequestStatus(statusStr)
	if err != nil {
		slog.ErrorContext(ctx, "corrupted status in email_change_requests table", "id", id, "status", statusStr, "error", err)
		return nil, apperror.NewInternal()
	}
	req := domainUser.RestoreEmailChangeRequest(id, returnedUserID, parsedEmail, codeStr, status, expiresAt, createdAt, updatedAt)

	return req, nil
}

func (r *EmailChangeRepository) InvalidatePendingByUserID(ctx context.Context, userID string, now time.Time) error {
	query := `
		UPDATE email_change_requests
		SET status = $1, updated_at = $2
		WHERE user_id = $3 AND status = $4`

	querier := infraPostgres.GetQuerier(ctx, r.querier)
	_, err := querier.Exec(ctx, query,
		domainUser.RequestStatusExpired.String(),
		now,
		userID,
		domainUser.RequestStatusPending.String(),
	)
	if err != nil {
		slog.ErrorContext(ctx, "database error invalidating pending email change requests", "user_id", userID, "error", err)
		return apperror.NewInternal()
	}
	return nil
}
