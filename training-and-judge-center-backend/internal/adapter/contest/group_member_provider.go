package contest

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GroupMemberProvider struct {
	db infraPostgres.Querier
}

func NewGroupMemberProvider(db *pgxpool.Pool) *GroupMemberProvider {
	return &GroupMemberProvider{db: db}
}

func (p *GroupMemberProvider) IsLeadOfGroup(ctx context.Context, userID, groupID string) (bool, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)
	var exists bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM group_members
			WHERE group_id=$1 AND user_id=$2 AND member_role='LEAD'
		)`, groupID, userID).Scan(&exists)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check lead membership", "error", err, "user_id", userID, "group_id", groupID)
		return false, apperror.NewInternal()
	}
	return exists, nil
}
