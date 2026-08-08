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

type RefreshTokenRepository struct {
	db infraPostgres.Querier
}

func NewRefreshTokenRepository(db infraPostgres.Querier) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Save(ctx context.Context, token *domainUser.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens
		(id, user_id, family_id, token_hash, issued_at, absolute_expires_at, user_agent, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	q := infraPostgres.GetQuerier(ctx, r.db)
	_, err := q.Exec(ctx, query,
		token.ID(),
		token.UserID(),
		token.FamilyID(),
		[]byte(token.TokenHash()),
		token.IssuedAt(),
		token.AbsoluteExpiresAt(),
		token.UserAgent(),
		token.IPAddress(),
	)
	if err != nil {
		slog.ErrorContext(ctx, "database error saving refresh token", "user_id", token.UserID(), "error", err)
		return apperror.NewInternal()
	}
	return nil
}

func (r *RefreshTokenRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domainUser.RefreshToken, error) {
	query := `
		SELECT id, user_id, family_id, token_hash, issued_at, absolute_expires_at,
		       revoked_at, replaced_by_id, user_agent, host(ip_address)
		FROM refresh_tokens
		WHERE token_hash = $1`

	q := infraPostgres.GetQuerier(ctx, r.db)
	return scanRefreshToken(ctx, q.QueryRow(ctx, query, []byte(tokenHash)))
}

func (r *RefreshTokenRepository) FindActiveByFamilyID(ctx context.Context, familyID string) (*domainUser.RefreshToken, error) {
	query := `
		SELECT id, user_id, family_id, token_hash, issued_at, absolute_expires_at,
		       revoked_at, replaced_by_id, user_agent, host(ip_address)
		FROM refresh_tokens
		WHERE family_id = $1 AND revoked_at IS NULL`

	q := infraPostgres.GetQuerier(ctx, r.db)
	return scanRefreshToken(ctx, q.QueryRow(ctx, query, familyID))
}

func scanRefreshToken(ctx context.Context, row pgx.Row) (*domainUser.RefreshToken, error) {
	var id, userID, familyID string
	var tokenHash []byte
	var issuedAt, absoluteExpiresAt time.Time
	var revokedAt *time.Time
	var replacedByID, userAgent, ipAddress *string

	err := row.Scan(
		&id,
		&userID,
		&familyID,
		&tokenHash,
		&issuedAt,
		&absoluteExpiresAt,
		&revokedAt,
		&replacedByID,
		&userAgent,
		&ipAddress,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "database error scanning refresh token", "error", err)
		return nil, apperror.NewInternal()
	}

	return domainUser.RestoreRefreshToken(
		id, userID, familyID, string(tokenHash),
		issuedAt, absoluteExpiresAt,
		revokedAt, replacedByID, userAgent, ipAddress,
	), nil
}

func (r *RefreshTokenRepository) Rotate(ctx context.Context, oldTokenHash string, newToken *domainUser.RefreshToken) (bool, error) {
	query := `
		WITH revoked AS (
			UPDATE refresh_tokens
			SET revoked_at = $1, replaced_by_id = $2
			WHERE token_hash = $3 AND revoked_at IS NULL
			RETURNING id
		)
		INSERT INTO refresh_tokens
			(id, user_id, family_id, token_hash, issued_at, absolute_expires_at, user_agent, ip_address)
		SELECT $4, $5, $6, $7, $8, $9, $10, $11
		FROM revoked`

	q := infraPostgres.GetQuerier(ctx, r.db)
	tag, err := q.Exec(ctx, query,
		newToken.IssuedAt(),
		newToken.ID(),
		[]byte(oldTokenHash),
		newToken.ID(),
		newToken.UserID(),
		newToken.FamilyID(),
		[]byte(newToken.TokenHash()),
		newToken.IssuedAt(),
		newToken.AbsoluteExpiresAt(),
		newToken.UserAgent(),
		newToken.IPAddress(),
	)
	if err != nil {
		slog.ErrorContext(ctx, "database error rotating refresh token", "family_id", newToken.FamilyID(), "error", err)
		return false, apperror.NewInternal()
	}
	return tag.RowsAffected() == 1, nil
}

func (r *RefreshTokenRepository) RevokeByFamilyID(ctx context.Context, familyID string, now time.Time) error {
	query := `UPDATE refresh_tokens SET revoked_at = $1 WHERE family_id = $2 AND revoked_at IS NULL`

	q := infraPostgres.GetQuerier(ctx, r.db)
	_, err := q.Exec(ctx, query, now, familyID)
	if err != nil {
		slog.ErrorContext(ctx, "database error revoking refresh token family", "family_id", familyID, "error", err)
		return apperror.NewInternal()
	}
	return nil
}

func (r *RefreshTokenRepository) RevokeAllByUserID(ctx context.Context, userID string, now time.Time) error {
	query := `UPDATE refresh_tokens SET revoked_at = $1 WHERE user_id = $2 AND revoked_at IS NULL`

	q := infraPostgres.GetQuerier(ctx, r.db)
	_, err := q.Exec(ctx, query, now, userID)
	if err != nil {
		slog.ErrorContext(ctx, "database error revoking refresh tokens for user", "user_id", userID, "error", err)
		return apperror.NewInternal()
	}
	return nil
}

func (r *RefreshTokenRepository) DeleteRevokedOrExpiredBefore(ctx context.Context, cutoff time.Time) error {
	query := `
		DELETE FROM refresh_tokens
		WHERE (revoked_at IS NOT NULL AND revoked_at < $1)
		   OR (absolute_expires_at < $1)`

	q := infraPostgres.GetQuerier(ctx, r.db)
	_, err := q.Exec(ctx, query, cutoff)
	if err != nil {
		slog.ErrorContext(ctx, "database error deleting old refresh tokens", "error", err)
		return apperror.NewInternal()
	}
	return nil
}
