package material

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type AuthorProvider struct {
	db infraPostgres.Querier
}

func NewAuthorProvider(db infraPostgres.Querier) *AuthorProvider {
	return &AuthorProvider{db: db}
}

func (p *AuthorProvider) GetDisplays(ctx context.Context, userIDs []string) (map[string]*appMaterial.AuthorDisplay, error) {
	if len(userIDs) == 0 {
		return map[string]*appMaterial.AuthorDisplay{}, nil
	}
	q := infraPostgres.GetQuerier(ctx, p.db)
	rows, err := q.Query(ctx, `SELECT id, nickname, name FROM users WHERE id = ANY($1)`, userIDs)
	if err != nil {
		slog.ErrorContext(ctx, "AuthorProvider.GetDisplays query failed", "error", err, "user_ids_count", len(userIDs))
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	out := make(map[string]*appMaterial.AuthorDisplay, len(userIDs))
	for rows.Next() {
		var id, nickname, name string
		if err := rows.Scan(&id, &nickname, &name); err != nil {
			slog.ErrorContext(ctx, "AuthorProvider.GetDisplays scan failed", "error", err, "user_id", id)
			return nil, apperror.NewInternal()
		}
		out[id] = &appMaterial.AuthorDisplay{Nickname: nickname, Name: name}
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "AuthorProvider.GetDisplays rows iteration error", "error", err, "user_ids_count", len(userIDs))
		return nil, apperror.NewInternal()
	}
	return out, nil
}
