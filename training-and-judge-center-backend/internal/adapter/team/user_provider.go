package team

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type UserProvider struct {
	db *pgxpool.Pool
}

func NewUserProvider(db *pgxpool.Pool) *UserProvider {
	return &UserProvider{db: db}
}

func (p *UserProvider) GetDisplay(ctx context.Context, userID string) (*appTeam.UserDisplay, error) {
	var nickname string
	err := p.db.QueryRow(ctx, `SELECT nickname FROM users WHERE id = $1`, userID).Scan(&nickname)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "GetDisplay query failed", "error", err, "user_id", userID)
		return nil, apperror.NewInternal()
	}
	return &appTeam.UserDisplay{Nickname: nickname}, nil
}
