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
