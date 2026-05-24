package problem

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type UserProvider struct {
	db *pgxpool.Pool
}

func NewUserProvider(db *pgxpool.Pool) *UserProvider {
	return &UserProvider{db: db}
}

func (p *UserProvider) ExistsByID(ctx context.Context, userID string) (bool, error) {
	var exists bool
	err := p.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&exists)
	if err != nil {
		slog.ErrorContext(ctx, "database error checking user existence", "user_id", userID, "error", err)
		return false, apperror.NewInternal()
	}
	return exists, nil
}

func (p *UserProvider) GetDisplay(ctx context.Context, userID string) (*appProblem.UserDisplay, error) {
	var nickname, name string
	err := p.db.QueryRow(ctx, `SELECT nickname, name FROM users WHERE id = $1`, userID).Scan(&nickname, &name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &appProblem.UserDisplay{Nickname: "unknown", Name: ""}, nil
		}
		slog.ErrorContext(ctx, "database error fetching user display", "user_id", userID, "error", err)
		return nil, apperror.NewInternal()
	}
	return &appProblem.UserDisplay{Nickname: nickname, Name: name}, nil
}

func (p *UserProvider) GetDisplays(ctx context.Context, userIDs []string) (map[string]*appProblem.UserDisplay, error) {
	if len(userIDs) == 0 {
		return map[string]*appProblem.UserDisplay{}, nil
	}
	rows, err := p.db.Query(ctx, `SELECT id, nickname, name FROM users WHERE id = ANY($1)`, userIDs)
	if err != nil {
		slog.ErrorContext(ctx, "database error fetching user displays", "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	result := make(map[string]*appProblem.UserDisplay, len(userIDs))
	for rows.Next() {
		var id, nickname, name string
		if err := rows.Scan(&id, &nickname, &name); err != nil {
			slog.ErrorContext(ctx, "database error scanning user display row", "error", err)
			return nil, apperror.NewInternal()
		}
		result[id] = &appProblem.UserDisplay{Nickname: nickname, Name: name}
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "database error iterating user display rows", "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}

func (p *UserProvider) GetIDByNickname(ctx context.Context, nickname string) (string, bool, error) {
	var id string
	err := p.db.QueryRow(ctx, `SELECT id FROM users WHERE LOWER(nickname) = LOWER($1)`, nickname).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		slog.ErrorContext(ctx, "database error fetching user id by nickname", "nickname", nickname, "error", err)
		return "", false, apperror.NewInternal()
	}
	return id, true, nil
}
