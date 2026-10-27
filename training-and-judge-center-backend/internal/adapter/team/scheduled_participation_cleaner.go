package team

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ScheduledParticipationCleaner struct {
	db infraPostgres.Querier
}

func NewScheduledParticipationCleaner(db infraPostgres.Querier) *ScheduledParticipationCleaner {
	return &ScheduledParticipationCleaner{db: db}
}

// RemoveUserFromScheduledParticipations removes userID from the selected_members array of every team
// registration for SCHEDULED (not yet started) contests. A registration the removal would leave below
// the contest's team_size_min is deleted instead: keeping it would leave the team enrolled with fewer
// players than the contest allows, and nobody would be told. The cancelled ones are returned.
func (s *ScheduledParticipationCleaner) RemoveUserFromScheduledParticipations(ctx context.Context, teamID, userID string) ([]appTeam.RegistrationAtRisk, error) {
	rows, err := infraPostgres.GetQuerier(ctx, s.db).Query(ctx, `
		WITH affected AS (
			SELECT ctp.contest_id, co.name, co.group_id, co.team_size_min,
			       cardinality(ctp.selected_members) - 1 AS remaining
			FROM   contest_team_participants ctp
			JOIN   contests co ON co.id = ctp.contest_id
			WHERE  ctp.team_id   = $1
			  AND  $2::uuid = ANY(ctp.selected_members)
			  AND  co.start_time > NOW()
		), cancelled AS (
			DELETE FROM contest_team_participants ctp
			USING  affected a
			WHERE  ctp.team_id = $1 AND ctp.contest_id = a.contest_id
			  AND  a.remaining < a.team_size_min
			RETURNING ctp.contest_id
		), reduced AS (
			UPDATE contest_team_participants ctp
			SET    selected_members = array_remove(ctp.selected_members, $2::uuid)
			FROM   affected a
			WHERE  ctp.team_id = $1 AND ctp.contest_id = a.contest_id
			  AND  a.remaining >= a.team_size_min
			RETURNING ctp.contest_id
		)
		SELECT a.contest_id::text, a.name, a.group_id::text
		FROM   affected a
		WHERE  a.remaining < a.team_size_min
		ORDER  BY a.name
	`, teamID, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to clean team selected_members on leave",
			"team_id", teamID, "user_id", userID, "error", err)
		return nil, apperror.NewInternal()
	}
	return scanRegistrationsAtRisk(ctx, rows, "clean team selected_members on leave")
}

// RegistrationsLosingMinimum lists the registrations RemoveUserFromScheduledParticipations would cancel.
func (s *ScheduledParticipationCleaner) RegistrationsLosingMinimum(ctx context.Context, teamID, userID string) ([]appTeam.RegistrationAtRisk, error) {
	rows, err := infraPostgres.GetQuerier(ctx, s.db).Query(ctx, `
		SELECT co.id::text, co.name, co.group_id::text
		FROM   contest_team_participants ctp
		JOIN   contests co ON co.id = ctp.contest_id
		WHERE  ctp.team_id   = $1
		  AND  $2::uuid = ANY(ctp.selected_members)
		  AND  co.start_time > NOW()
		  AND  cardinality(ctp.selected_members) - 1 < co.team_size_min
		ORDER  BY co.name
	`, teamID, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list registrations a leave would cancel",
			"team_id", teamID, "user_id", userID, "error", err)
		return nil, apperror.NewInternal()
	}
	return scanRegistrationsAtRisk(ctx, rows, "list registrations a leave would cancel")
}

type registrationRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}

func scanRegistrationsAtRisk(ctx context.Context, rows registrationRows, what string) ([]appTeam.RegistrationAtRisk, error) {
	defer rows.Close()
	out := []appTeam.RegistrationAtRisk{}
	for rows.Next() {
		var r appTeam.RegistrationAtRisk
		if err := rows.Scan(&r.ContestID, &r.ContestName, &r.GroupID); err != nil {
			slog.ErrorContext(ctx, "failed to scan registration: "+what, "error", err)
			return nil, apperror.NewInternal()
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "registration rows error: "+what, "error", err)
		return nil, apperror.NewInternal()
	}
	return out, nil
}
