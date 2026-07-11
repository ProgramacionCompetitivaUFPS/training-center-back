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

type EmailResolver struct {
	db *pgxpool.Pool
}

func NewEmailResolver(db *pgxpool.Pool) *EmailResolver {
	return &EmailResolver{db: db}
}

// ResolveByEmail returns (nil, nil) when no user has that email.
func (r *EmailResolver) ResolveByEmail(ctx context.Context, email string) (*appGroup.UserDisplay, error) {
	const q = `SELECT id, nickname, name, email, role FROM users WHERE email = LOWER($1) LIMIT 1`
	var id, nick, name, mail, role string
	err := r.db.QueryRow(ctx, q, email).Scan(&id, &nick, &name, &mail, &role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "EmailResolver.ResolveByEmail failed", "error", err)
		return nil, apperror.NewInternal()
	}
	return &appGroup.UserDisplay{
		ID:         id,
		Nickname:   nick,
		Name:       name,
		Email:      mail,
		SystemRole: role,
	}, nil
}
