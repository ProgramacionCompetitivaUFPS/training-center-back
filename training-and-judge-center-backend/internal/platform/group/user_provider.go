package group

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
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
	rows, err := p.db.Query(ctx, `SELECT id, nickname, name FROM users WHERE id = ANY($1)`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]*appGroup.UserDisplay, len(userIDs))
	for rows.Next() {
		var id, nickname, name string
		if err := rows.Scan(&id, &nickname, &name); err != nil {
			return nil, err
		}
		out[id] = &appGroup.UserDisplay{Nickname: nickname, Name: name}
	}
	return out, rows.Err()
}
