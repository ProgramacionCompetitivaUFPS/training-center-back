package submission

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type UserProvider struct {
	db infraPostgres.Querier
}

func NewUserProvider(db infraPostgres.Querier) *UserProvider {
	return &UserProvider{db: db}
}

func (p *UserProvider) GetUserByID(ctx context.Context, userID string) (*appsubmission.UserDisplay, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	var id, nickname string
	err := q.QueryRow(ctx, `SELECT id, nickname FROM users WHERE id = $1`, userID).Scan(&id, &nickname)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound(domainuser.ErrCodeUserNotFound, "user not found")
		}
		slog.ErrorContext(ctx, "submission: failed to get user display", "user_id", userID, "error", err)
		return nil, apperror.NewInternal()
	}

	return &appsubmission.UserDisplay{ID: id, Nickname: nickname}, nil
}
