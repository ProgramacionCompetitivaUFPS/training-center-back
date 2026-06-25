package material

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/training-judge-center/backend/pkg/apperror"
)

type AuthorIDProvider struct {
	db *pgxpool.Pool
}

func NewAuthorIDProvider(db *pgxpool.Pool) *AuthorIDProvider {
	return &AuthorIDProvider{db: db}
}

func (p *AuthorIDProvider) FindIDByNickname(ctx context.Context, nickname string) (string, bool, error) {
	var id string
	err := p.db.QueryRow(ctx, `SELECT id FROM users WHERE LOWER(nickname) = LOWER($1)`, nickname).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		slog.ErrorContext(ctx, "author_id_provider: failed to find user by nickname", "nickname", nickname, "error", err)
		return "", false, apperror.NewInternal()
	}
	return id, true, nil
}
