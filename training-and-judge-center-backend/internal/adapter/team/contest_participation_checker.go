package team

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ContestParticipationChecker checks whether a user appears in the
// selected_members of any contest_team_participants row for an ACTIVE contest.
type ContestParticipationChecker struct {
	db infraPostgres.Querier
}

func NewContestParticipationChecker(db infraPostgres.Querier) *ContestParticipationChecker {
	return &ContestParticipationChecker{db: db}
}

func (c *ContestParticipationChecker) IsUserInActiveContestForTeam(ctx context.Context, teamID, userID string) (bool, error) {
	q := infraPostgres.GetQuerier(ctx, c.db)

	var exists bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM contest_team_participants ctp
			JOIN contests co ON co.id = ctp.contest_id
			WHERE ctp.team_id = $1
			  AND $2 = ANY(ctp.selected_members)
			  AND co.start_time <= NOW()
			  AND co.end_time > NOW()
		)`, teamID, userID,
	).Scan(&exists)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check contest participation",
			"team_id", teamID, "user_id", userID, "error", err)
		return false, apperror.NewInternal()
	}
	return exists, nil
}
