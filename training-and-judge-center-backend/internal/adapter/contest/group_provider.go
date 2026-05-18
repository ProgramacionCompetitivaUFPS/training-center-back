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

type GroupProvider struct {
	db *pgxpool.Pool
}

func NewGroupProvider(db *pgxpool.Pool) *GroupProvider {
	return &GroupProvider{db: db}
}

func (p *GroupProvider) FindByID(ctx context.Context, groupID string) (*appContest.GroupInfo, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)
	var id, name string
	err := q.QueryRow(ctx, `SELECT id, name FROM groups WHERE id=$1`, groupID).Scan(&id, &name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "failed to find group for contest", "error", err, "group_id", groupID)
		return nil, apperror.NewInternal()
	}
	return &appContest.GroupInfo{ID: id, Name: name}, nil
}
