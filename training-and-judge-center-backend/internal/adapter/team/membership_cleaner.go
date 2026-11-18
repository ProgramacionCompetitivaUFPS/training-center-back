package team

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// MembershipCleaner removes a user from every team when their account is deactivated.
type MembershipCleaner struct {
	db        infraPostgres.Querier
	scheduled *ScheduledParticipationCleaner
}

func NewMembershipCleaner(db infraPostgres.Querier) *MembershipCleaner {
	return &MembershipCleaner{db: db, scheduled: NewScheduledParticipationCleaner(db)}
}

// RemoveUserFromAllTeams must run inside the caller's transaction: it uses the querier in ctx, so the
// deactivation and the cleanup commit or roll back together.
func (c *MembershipCleaner) RemoveUserFromAllTeams(ctx context.Context, userID string) error {
	q := infraPostgres.GetQuerier(ctx, c.db)

	rows, err := q.Query(ctx, `SELECT team_id::text FROM team_members WHERE user_id = $1`, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list the teams of a deactivated user", "user_id", userID, "error", err)
		return apperror.NewInternal()
	}
	var teamIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			slog.ErrorContext(ctx, "failed to scan a team of a deactivated user", "user_id", userID, "error", err)
			return apperror.NewInternal()
		}
		teamIDs = append(teamIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "failed to read the teams of a deactivated user", "user_id", userID, "error", err)
		return apperror.NewInternal()
	}

	for _, teamID := range teamIDs {
		if _, err := c.scheduled.RemoveUserFromScheduledParticipations(ctx, teamID, userID); err != nil {
			return err
		}
	}

	if _, err := q.Exec(ctx, `DELETE FROM team_members WHERE user_id = $1`, userID); err != nil {
		slog.ErrorContext(ctx, "failed to delete the team memberships of a deactivated user", "user_id", userID, "error", err)
		return apperror.NewInternal()
	}
	if _, err := q.Exec(ctx, `DELETE FROM team_invitations WHERE invitee_id = $1`, userID); err != nil {
		slog.ErrorContext(ctx, "failed to delete the team invitations of a deactivated user", "user_id", userID, "error", err)
		return apperror.NewInternal()
	}
	return nil
}
