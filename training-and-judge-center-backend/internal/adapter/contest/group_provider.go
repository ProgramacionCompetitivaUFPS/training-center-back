package contest

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appContest "github.com/training-judge-center/backend/internal/application/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GroupProvider struct {
	db infraPostgres.Querier
}

func NewGroupProvider(db infraPostgres.Querier) *GroupProvider {
	return &GroupProvider{db: db}
}

func (p *GroupProvider) FindByID(ctx context.Context, groupID string) (*appContest.GroupInfo, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)
	var id, name, visibility string
	err := q.QueryRow(ctx, `SELECT id, name, visibility FROM groups WHERE id=$1`, groupID).Scan(&id, &name, &visibility)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "failed to find group for contest", "error", err, "group_id", groupID)
		return nil, apperror.NewInternal()
	}
	return &appContest.GroupInfo{ID: id, Name: name, IsVisible: visibility == "VISIBLE"}, nil
}

func (p *GroupProvider) FindByIDs(ctx context.Context, groupIDs []string) (map[string]*appContest.GroupInfo, error) {
	result := make(map[string]*appContest.GroupInfo, len(groupIDs))
	if len(groupIDs) == 0 {
		return result, nil
	}

	q := infraPostgres.GetQuerier(ctx, p.db)
	rows, err := q.Query(ctx, `SELECT id, name, visibility FROM groups WHERE id = ANY($1)`, groupIDs)
	if err != nil {
		slog.ErrorContext(ctx, "failed to bulk find groups for contests", "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	for rows.Next() {
		var id, name, visibility string
		if err := rows.Scan(&id, &name, &visibility); err != nil {
			slog.ErrorContext(ctx, "failed to scan group row", "error", err)
			return nil, apperror.NewInternal()
		}
		result[id] = &appContest.GroupInfo{ID: id, Name: name, IsVisible: visibility == "VISIBLE"}
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "bulk find groups rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}

func (p *GroupProvider) ListAccessibleGroupIDs(ctx context.Context, viewerID string, isAdmin bool) ([]string, error) {
	if isAdmin {
		return nil, nil
	}

	q := infraPostgres.GetQuerier(ctx, p.db)
	rows, err := q.Query(ctx, `
		SELECT id FROM groups
		WHERE visibility = 'VISIBLE'
		   OR EXISTS (SELECT 1 FROM group_members gm WHERE gm.group_id = groups.id AND gm.user_id = $1)`,
		viewerID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list accessible group ids", "error", err, "viewer_id", viewerID)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			slog.ErrorContext(ctx, "failed to scan group id", "error", err)
			return nil, apperror.NewInternal()
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "list accessible group ids rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return ids, nil
}
