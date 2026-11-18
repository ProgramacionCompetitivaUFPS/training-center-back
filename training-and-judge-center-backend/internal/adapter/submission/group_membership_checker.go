package submission

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GroupMembershipChecker struct {
	db infraPostgres.Querier
}

func NewGroupMembershipChecker(db infraPostgres.Querier) *GroupMembershipChecker {
	return &GroupMembershipChecker{db: db}
}

func (c *GroupMembershipChecker) IsGroupMember(ctx context.Context, groupID, userID string) (bool, error) {
	var exists bool
	err := infraPostgres.GetQuerier(ctx, c.db).QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)`,
		groupID, userID,
	).Scan(&exists)
	if err != nil {
		slog.ErrorContext(ctx, "submission: failed to check group membership",
			"group_id", groupID, "user_id", userID, "error", err)
		return false, apperror.NewInternal()
	}
	return exists, nil
}
