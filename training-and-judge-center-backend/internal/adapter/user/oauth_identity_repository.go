package user

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	domainUser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var _ domainUser.OAuthIdentityRepository = (*OAuthIdentityRepository)(nil)

type OAuthIdentityRepository struct {
	db infraPostgres.Querier
}

func NewOAuthIdentityRepository(db infraPostgres.Querier) *OAuthIdentityRepository {
	return &OAuthIdentityRepository{db: db}
}

func (r *OAuthIdentityRepository) Save(ctx context.Context, identity *domainUser.OAuthIdentity) error {
	query := `
		INSERT INTO oauth_identities (id, user_id, provider, provider_user_id, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	q := infraPostgres.GetQuerier(ctx, r.db)
	_, err := q.Exec(ctx, query,
		identity.ID(),
		identity.UserID(),
		identity.Provider().String(),
		identity.ProviderUserID(),
		identity.CreatedAt(),
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == infraPostgres.UniqueViolation {
			switch pgErr.ConstraintName {
			case "idx_oauth_identities_provider_user":
				return apperror.NewConflict(domainUser.ErrCodeOAuthIdentityConflict, "this Google account is already linked to a user")
			case "idx_oauth_identities_user_provider":
				return apperror.NewConflict(domainUser.ErrCodeOAuthIdentityAlreadyLinked, "this user already has a Google identity linked")
			}
		}
		slog.ErrorContext(ctx, "database error saving oauth identity", "user_id", identity.UserID(), "error", err)
		return apperror.NewInternal()
	}
	return nil
}

func (r *OAuthIdentityRepository) FindByProvider(ctx context.Context, provider domainUser.OAuthProvider, providerUserID string) (*domainUser.OAuthIdentity, error) {
	query := `
		SELECT id, user_id, created_at
		FROM oauth_identities
		WHERE provider = $1 AND provider_user_id = $2`

	var id, userID string
	var createdAt time.Time

	q := infraPostgres.GetQuerier(ctx, r.db)
	err := q.QueryRow(ctx, query, provider.String(), providerUserID).Scan(&id, &userID, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "database error in FindByProvider", "provider", provider.String(), "error", err)
		return nil, apperror.NewInternal()
	}

	return domainUser.RestoreOAuthIdentity(id, userID, provider, providerUserID, createdAt), nil
}

func (r *OAuthIdentityRepository) FindByUserID(ctx context.Context, userID string, provider domainUser.OAuthProvider) (*domainUser.OAuthIdentity, error) {
	query := `
		SELECT id, provider_user_id, created_at
		FROM oauth_identities
		WHERE user_id = $1 AND provider = $2`

	var id, providerUserID string
	var createdAt time.Time

	q := infraPostgres.GetQuerier(ctx, r.db)
	err := q.QueryRow(ctx, query, userID, provider.String()).Scan(&id, &providerUserID, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "database error in FindByUserID", "user_id", userID, "error", err)
		return nil, apperror.NewInternal()
	}

	return domainUser.RestoreOAuthIdentity(id, userID, provider, providerUserID, createdAt), nil
}

func (r *OAuthIdentityRepository) DeleteByUserID(ctx context.Context, userID string, provider domainUser.OAuthProvider) (bool, error) {
	query := `DELETE FROM oauth_identities WHERE user_id = $1 AND provider = $2`

	q := infraPostgres.GetQuerier(ctx, r.db)
	tag, err := q.Exec(ctx, query, userID, provider.String())
	if err != nil {
		slog.ErrorContext(ctx, "database error deleting oauth identity", "user_id", userID, "error", err)
		return false, apperror.NewInternal()
	}
	return tag.RowsAffected() > 0, nil
}
