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
// A registration the removal would leave below the contest's team_size_min is deleted
// instead, as ScheduledParticipationCleaner does when a member leaves the team. The rows
// are locked first so two concurrent removals count the members that really remain.
func (c *TeamSelectionCleaner) RemoveFromScheduledByGroupAndUser(ctx context.Context, groupID, userID string) error {
	_, err := infraPostgres.GetQuerier(ctx, c.db).Exec(ctx, `
		WITH affected AS (
			SELECT ctp.id, co.team_size_min,
			       cardinality(ctp.selected_members) - 1 AS remaining
			FROM   contest_team_participants ctp
			JOIN   contests co ON co.id = ctp.contest_id
			WHERE  co.group_id = $1
			  AND  co.start_time > NOW()
			  AND  $2::uuid = ANY(ctp.selected_members)
			FOR UPDATE OF ctp
		), cancelled AS (
			DELETE FROM contest_team_participants ctp
			USING  affected a
			WHERE  ctp.id = a.id AND a.remaining < a.team_size_min
		), reduced AS (
			UPDATE contest_team_participants ctp
			SET    selected_members = array_remove(ctp.selected_members, $2::uuid)
			FROM   affected a
			WHERE  ctp.id = a.id AND a.remaining >= a.team_size_min
		)
		SELECT 1
	`, groupID, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to remove team selection on member removal",
			"group_id", groupID, "user_id", userID, "error", err)
		return apperror.NewInternal()
	}
	return nil
}
