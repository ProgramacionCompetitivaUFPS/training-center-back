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

// IsTeammate reports whether viewer and author played the contest together: both are selected in the
// same team registration, and the viewer is still on the team.
//
// The author's team membership is deliberately not required. selected_members is the roster of the
// contest; the author may no longer be on the team (a deactivated account is taken off its teams), and
// the people they played with must keep seeing the code they wrote together.
func (c *TeamMembershipChecker) IsTeammate(ctx context.Context, contestID, viewerID, authorID string) (bool, error) {
	q := infraPostgres.GetQuerier(ctx, c.db)

	var exists bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM contest_team_participants ctp
			JOIN team_members tm1 ON tm1.team_id = ctp.team_id AND tm1.user_id = $2
			WHERE ctp.contest_id = $1
			  AND ctp.selected_members @> ARRAY[$2::uuid]
			  AND ctp.selected_members @> ARRAY[$3::uuid]
		)
	`, contestID, viewerID, authorID).Scan(&exists)
	if err != nil {
		slog.ErrorContext(ctx, "submission: failed to check team membership",
			"contest_id", contestID, "viewer_id", viewerID, "author_id", authorID, "error", err)
		return false, apperror.NewInternal()
	}

	return exists, nil
}
