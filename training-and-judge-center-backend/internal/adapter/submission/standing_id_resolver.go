package submission

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type StandingIDResolver struct {
	db infraPostgres.Querier
}

func NewStandingIDResolver(db infraPostgres.Querier) *StandingIDResolver {
	return &StandingIDResolver{db: db}
}

// ResolveStandingID checks first if the user has an individual registration,
// then checks if the user is in the selectedMembers of any registered team.
// Returns (userID, true, nil), (teamID, true, nil), or ("", false, nil).
func (r *StandingIDResolver) ResolveStandingID(ctx context.Context, contestID, userID string) (string, bool, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)

	// First: individual registration (fast O(1) lookup)
	var exists bool
	if err := q.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM contest_registrations WHERE contest_id=$1 AND user_id=$2)`,
		contestID, userID,
	).Scan(&exists); err != nil {
		slog.ErrorContext(ctx, "submission: failed to check individual contest registration",
			"contest_id", contestID, "user_id", userID, "error", err)
		return "", false, apperror.NewInternal()
	}
	if exists {
		return userID, true, nil
	}

	// Second: team selectedMembers
	var teamID string
	err := q.QueryRow(ctx, `
		SELECT ctp.team_id
		FROM contest_team_participants ctp
		JOIN team_members tm ON tm.team_id = ctp.team_id AND tm.user_id = $2
		WHERE ctp.contest_id = $1
		  AND (
		    array_length(ctp.selected_members, 1) IS NULL
		    OR ctp.selected_members @> ARRAY[$2::uuid]
		  )
		LIMIT 1
	`, contestID, userID).Scan(&teamID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		slog.ErrorContext(ctx, "submission: failed to check team contest participation",
			"contest_id", contestID, "user_id", userID, "error", err)
		return "", false, apperror.NewInternal()
	}

	return teamID, true, nil
}
