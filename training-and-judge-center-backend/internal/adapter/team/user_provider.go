package team

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type UserProvider struct {
	db infraPostgres.Querier
}

func NewUserProvider(db infraPostgres.Querier) *UserProvider {
	return &UserProvider{db: db}
}

func (p *UserProvider) GetDisplay(ctx context.Context, userID string) (*appTeam.UserDisplay, error) {
	var nickname string
	err := infraPostgres.GetQuerier(ctx, p.db).QueryRow(ctx, `SELECT nickname FROM users WHERE id = $1`, userID).Scan(&nickname)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "GetDisplay query failed", "error", err, "user_id", userID)
		return nil, apperror.NewInternal()
	}
	return &appTeam.UserDisplay{Nickname: nickname}, nil
}

func (p *UserProvider) GetDisplays(ctx context.Context, userIDs []string) (map[string]*appTeam.UserDisplay, error) {
	if len(userIDs) == 0 {
		return map[string]*appTeam.UserDisplay{}, nil
	}

	const q = `SELECT id, nickname FROM users WHERE id = ANY($1)`
	rows, err := infraPostgres.GetQuerier(ctx, p.db).Query(ctx, q, userIDs)
	if err != nil {
		slog.ErrorContext(ctx, "GetDisplays query failed", "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	result := make(map[string]*appTeam.UserDisplay, len(userIDs))
	for rows.Next() {
		var id, nickname string
		if err := rows.Scan(&id, &nickname); err != nil {
			slog.ErrorContext(ctx, "GetDisplays scan failed", "error", err)
			return nil, apperror.NewInternal()
		}
		result[id] = &appTeam.UserDisplay{ID: id, Nickname: nickname}
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "GetDisplays rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}

func (p *UserProvider) FindByNickname(ctx context.Context, nickname string) (*appTeam.UserDisplay, error) {
	const q = `SELECT id, nickname FROM users WHERE nickname = $1`
	var id, nick string
	err := infraPostgres.GetQuerier(ctx, p.db).QueryRow(ctx, q, nickname).Scan(&id, &nick)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound(domainTeam.ErrCodeUserNotFound, "User not found")
		}
		slog.ErrorContext(ctx, "FindByNickname query failed", "error", err, "nickname", nickname)
		return nil, apperror.NewInternal()
	}
	return &appTeam.UserDisplay{ID: id, Nickname: nick}, nil
}
