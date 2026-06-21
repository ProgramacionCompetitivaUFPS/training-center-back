package submission

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type TeamMembershipChecker struct {
	db infraPostgres.Querier
}

func NewTeamMembershipChecker(db infraPostgres.Querier) *TeamMembershipChecker {
	return &TeamMembershipChecker{db: db}
}

// empty selected_members means all team members count; non-empty restricts to listed UUIDs
func (c *TeamMembershipChecker) IsTeammate(ctx context.Context, contestID, viewerID, authorID string) (bool, error) {
	q := infraPostgres.GetQuerier(ctx, c.db)

	var exists bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM contest_team_participants ctp
			JOIN team_members tm1 ON tm1.team_id = ctp.team_id AND tm1.user_id = $2
			JOIN team_members tm2 ON tm2.team_id = ctp.team_id AND tm2.user_id = $3
			WHERE ctp.contest_id = $1
			  AND (
			    array_length(ctp.selected_members, 1) IS NULL
			    OR (
			      ctp.selected_members @> ARRAY[$2::uuid]
			      AND ctp.selected_members @> ARRAY[$3::uuid]
			    )
			  )
		)
	`, contestID, viewerID, authorID).Scan(&exists)
	if err != nil {
		slog.ErrorContext(ctx, "submission: failed to check team membership",
			"contest_id", contestID, "viewer_id", viewerID, "author_id", authorID, "error", err)
		return false, apperror.NewInternal()
	}

	return exists, nil
}
