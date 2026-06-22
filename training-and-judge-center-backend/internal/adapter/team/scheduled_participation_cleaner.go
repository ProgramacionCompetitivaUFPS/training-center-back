package team

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ScheduledParticipationCleaner struct {
	db infraPostgres.Querier
}

func NewScheduledParticipationCleaner(db infraPostgres.Querier) *ScheduledParticipationCleaner {
	return &ScheduledParticipationCleaner{db: db}
}

// RemoveUserFromScheduledParticipations removes userID from the selected_members array
// of every team registration for SCHEDULED (not yet started) contests.
func (s *ScheduledParticipationCleaner) RemoveUserFromScheduledParticipations(ctx context.Context, teamID, userID string) (int, error) {
	tag, err := infraPostgres.GetQuerier(ctx, s.db).Exec(ctx, `
		UPDATE contest_team_participants ctp
		SET    selected_members = array_remove(ctp.selected_members, $2::uuid)
		FROM   contests co
		WHERE  ctp.contest_id = co.id
		  AND  ctp.team_id    = $1
		  AND  $2::uuid = ANY(ctp.selected_members)
		  AND  co.start_time  > NOW()
	`, teamID, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to clean team selected_members on leave",
			"team_id", teamID, "user_id", userID, "error", err)
		return 0, apperror.NewInternal()
	}
	return int(tag.RowsAffected()), nil
}
