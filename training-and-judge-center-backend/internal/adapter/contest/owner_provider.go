package contest

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appContest "github.com/training-judge-center/backend/internal/application/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type OwnerProvider struct {
	db *pgxpool.Pool
}

func NewOwnerProvider(db *pgxpool.Pool) *OwnerProvider {
	return &OwnerProvider{db: db}
}

func (p *OwnerProvider) GetDisplay(ctx context.Context, userID string) (*appContest.UserDisplay, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)
	var nickname, name string
	err := q.QueryRow(ctx, `SELECT nickname, name FROM users WHERE id=$1`, userID).Scan(&nickname, &name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "failed to get owner display", "error", err, "user_id", userID)
		return nil, apperror.NewInternal()
	}
	return &appContest.UserDisplay{Nickname: nickname, Name: name}, nil
}
