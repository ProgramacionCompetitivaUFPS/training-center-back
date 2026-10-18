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

func (p *UserProvider) GetUsersByIDs(ctx context.Context, userIDs []string) (map[string]*appsubmission.UserDisplay, error) {
	if len(userIDs) == 0 {
		return map[string]*appsubmission.UserDisplay{}, nil
	}
	q := infraPostgres.GetQuerier(ctx, p.db)

	rows, err := q.Query(ctx, `SELECT id, nickname FROM users WHERE id = ANY($1)`, userIDs)
	if err != nil {
		slog.ErrorContext(ctx, "submission: failed to get user displays", "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	result := make(map[string]*appsubmission.UserDisplay, len(userIDs))
	for rows.Next() {
		var id, nickname string
		if err := rows.Scan(&id, &nickname); err != nil {
			slog.ErrorContext(ctx, "submission: failed to scan user display row", "error", err)
			return nil, apperror.NewInternal()
		}
		result[id] = &appsubmission.UserDisplay{ID: id, Nickname: nickname}
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "submission: error iterating user display rows", "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}
