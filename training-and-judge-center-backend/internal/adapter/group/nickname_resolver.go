package group

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type NicknameResolver struct {
	db *pgxpool.Pool
}

func NewNicknameResolver(db *pgxpool.Pool) *NicknameResolver {
	return &NicknameResolver{db: db}
}

// ResolveByNickname returns (nil, nil) when no user has that nickname.
func (r *NicknameResolver) ResolveByNickname(ctx context.Context, nickname string) (*appGroup.UserDisplay, error) {
	const q = `SELECT id, nickname, name, email, role FROM users WHERE nickname = LOWER($1) LIMIT 1`
	var id, nick, name, email, role string
	err := r.db.QueryRow(ctx, q, nickname).Scan(&id, &nick, &name, &email, &role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "NicknameResolver.ResolveByNickname failed", "nickname", nickname, "error", err)
		return nil, apperror.NewInternal()
	}
	return &appGroup.UserDisplay{
		ID:         id,
		Nickname:   nick,
		Name:       name,
		Email:      email,
		SystemRole: role,
	}, nil
}
