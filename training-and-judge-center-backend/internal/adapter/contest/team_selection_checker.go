package contest

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type TeamSelectionChecker struct {
	db infraPostgres.Querier
}

func NewTeamSelectionChecker(db infraPostgres.Querier) *TeamSelectionChecker {
	return &TeamSelectionChecker{db: db}
}

func (c *TeamSelectionChecker) IsUserSelectedInAnyTeam(ctx context.Context, contestID string, userID string) (bool, error) {
	q := infraPostgres.GetQuerier(ctx, c.db)
	var exists bool
	err := q.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM contest_team_participants
			WHERE contest_id=$1 AND $2::UUID = ANY(selected_members)
		)`,
		contestID, userID,
	).Scan(&exists)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check team selection",
			"contest_id", contestID, "user_id", userID, "error", err)
		return false, apperror.NewInternal()
	}
	return exists, nil
}
