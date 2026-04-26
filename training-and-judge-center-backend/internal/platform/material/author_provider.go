package material

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type AuthorProvider struct {
	db *pgxpool.Pool
}

func NewAuthorProvider(db *pgxpool.Pool) *AuthorProvider {
	return &AuthorProvider{db: db}
}

func (p *AuthorProvider) GetDisplays(ctx context.Context, userIDs []string) (map[string]*appMaterial.AuthorDisplay, error) {
	if len(userIDs) == 0 {
		return map[string]*appMaterial.AuthorDisplay{}, nil
	}
	rows, err := p.db.Query(ctx, `SELECT id, nickname, name FROM users WHERE id = ANY($1)`, userIDs)
	if err != nil {
		slog.ErrorContext(ctx, "AuthorProvider.GetDisplays query failed", "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	out := make(map[string]*appMaterial.AuthorDisplay, len(userIDs))
	for rows.Next() {
		var id, nickname, name string
		if err := rows.Scan(&id, &nickname, &name); err != nil {
			slog.ErrorContext(ctx, "AuthorProvider.GetDisplays scan failed", "error", err)
			return nil, apperror.NewInternal()
		}
		out[id] = &appMaterial.AuthorDisplay{Nickname: nickname, Name: name}
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "AuthorProvider.GetDisplays rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return out, nil
}
