package group

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type UserProvider struct {
	db *pgxpool.Pool
}

func NewUserProvider(db *pgxpool.Pool) *UserProvider {
	return &UserProvider{db: db}
}

func (p *UserProvider) GetDisplays(ctx context.Context, userIDs []string) (map[string]*appGroup.UserDisplay, error) {
	if len(userIDs) == 0 {
		return map[string]*appGroup.UserDisplay{}, nil
	}
	rows, err := infraPostgres.GetQuerier(ctx, p.db).Query(ctx, `SELECT id, nickname, name, email, role FROM users WHERE id = ANY($1)`, userIDs)
	if err != nil {
		slog.ErrorContext(ctx, "GetDisplays query failed", "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	out := make(map[string]*appGroup.UserDisplay, len(userIDs))
	for rows.Next() {
		var id, nickname, name, role string
		var email *string
		if err := rows.Scan(&id, &nickname, &name, &email, &role); err != nil {
			slog.ErrorContext(ctx, "GetDisplays scan failed", "error", err)
			return nil, apperror.NewInternal()
		}
		// email es NULL para cuentas desactivadas (ver User.Deactivate) — una
		// cuenta anonimizada sigue apareciendo en listados de miembros/líderes
		// aunque ya no tenga correo.
		var emailVal string
		if email != nil {
			emailVal = *email
		}
		out[id] = &appGroup.UserDisplay{ID: id, Nickname: nickname, Name: name, Email: emailVal, SystemRole: role}
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "GetDisplays rows failed", "error", err)
		return nil, apperror.NewInternal()
	}
	return out, nil
}
