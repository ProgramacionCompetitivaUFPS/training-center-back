package group

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type TeamSelectionCleaner struct {
	db infraPostgres.Querier
}

func NewTeamSelectionCleaner(db infraPostgres.Querier) *TeamSelectionCleaner {
	return &TeamSelectionCleaner{db: db}
}

// RemoveFromScheduledByGroupAndUser removes userID from selected_members in any
// SCHEDULED (not yet started) contest_team_participants row belonging to groupID.
func (c *TeamSelectionCleaner) RemoveFromScheduledByGroupAndUser(ctx context.Context, groupID, userID string) error {
	_, err := infraPostgres.GetQuerier(ctx, c.db).Exec(ctx, `
		UPDATE contest_team_participants
		SET selected_members = array_remove(selected_members, $2::UUID)
		WHERE contest_id IN (
			SELECT id FROM contests
			WHERE group_id = $1 AND start_time > NOW()
		)
		AND $2::UUID = ANY(selected_members)
	`, groupID, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to remove team selection on member removal",
			"group_id", groupID, "user_id", userID, "error", err)
		return apperror.NewInternal()
	}
	return nil
}
