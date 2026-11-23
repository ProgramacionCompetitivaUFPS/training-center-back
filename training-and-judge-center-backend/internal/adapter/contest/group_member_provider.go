package contest

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GroupMemberProvider struct {
	db infraPostgres.Querier
}

func NewGroupMemberProvider(db infraPostgres.Querier) *GroupMemberProvider {
	return &GroupMemberProvider{db: db}
}

func (p *GroupMemberProvider) GetMemberRoleBulk(ctx context.Context, userID string, groupIDs []string) (map[string]*string, error) {
	if len(groupIDs) == 0 {
		return map[string]*string{}, nil
	}
	q := infraPostgres.GetQuerier(ctx, p.db)
	rows, err := q.Query(ctx,
		`SELECT group_id, member_role FROM group_members WHERE user_id = $1 AND group_id = ANY($2)`,
		userID, groupIDs,
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to bulk get member roles", "error", err, "user_id", userID)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	result := make(map[string]*string, len(groupIDs))
	for rows.Next() {
		var groupID, role string
		if err := rows.Scan(&groupID, &role); err != nil {
			slog.ErrorContext(ctx, "failed to scan member role row", "error", err, "user_id", userID)
			return nil, apperror.NewInternal()
		}
		r := role
		result[groupID] = &r
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "row iteration error in bulk member roles", "error", err, "user_id", userID)
		return nil, apperror.NewInternal()
	}
	return result, nil
}

func (p *GroupMemberProvider) GetMemberRole(ctx context.Context, userID, groupID string) (*string, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)
	var role string
	err := q.QueryRow(ctx,
		`SELECT member_role FROM group_members WHERE group_id = $1 AND user_id = $2`,
		groupID, userID,
	).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "failed to get member role", "error", err, "user_id", userID, "group_id", groupID)
		return nil, apperror.NewInternal()
	}
	return &role, nil
}
