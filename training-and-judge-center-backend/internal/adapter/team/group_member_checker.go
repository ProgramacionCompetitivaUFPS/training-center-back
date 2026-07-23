package team

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GroupMemberChecker struct {
	db infraPostgres.Querier
}

func NewGroupMemberChecker(db infraPostgres.Querier) *GroupMemberChecker {
	return &GroupMemberChecker{db: db}
}

func (c *GroupMemberChecker) AreUsersGroupMembers(ctx context.Context, groupID string, userIDs []string) (map[string]bool, error) {
	result := make(map[string]bool, len(userIDs))
	for _, id := range userIDs {
		result[id] = false
	}
	if len(userIDs) == 0 {
		return result, nil
	}

	q := infraPostgres.GetQuerier(ctx, c.db)
	rows, err := q.Query(ctx,
		`SELECT user_id FROM group_members WHERE group_id=$1 AND user_id = ANY($2::UUID[])`,
		groupID, userIDs,
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check group memberships",
			"group_id", groupID, "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			slog.ErrorContext(ctx, "failed to scan group member row", "error", err)
			return nil, apperror.NewInternal()
		}
		result[userID] = true
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "group members rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}
