package contest

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appContest "github.com/training-judge-center/backend/internal/application/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type CallerStandingProvider struct {
	db infraPostgres.Querier
}

var _ appContest.CallerStandingProvider = (*CallerStandingProvider)(nil)

func NewCallerStandingProvider(db infraPostgres.Querier) *CallerStandingProvider {
	return &CallerStandingProvider{db: db}
}

func (p *CallerStandingProvider) GetCallerStandingID(ctx context.Context, contestID, userID string) (string, bool, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	var dummy string
	err := q.QueryRow(ctx,
		`SELECT id FROM contest_registrations WHERE contest_id = $1 AND user_id = $2`,
		contestID, userID,
	).Scan(&dummy)
	if err == nil {
		return userID, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		slog.ErrorContext(ctx, "contest: failed to check individual registration for standing",
			"contest_id", contestID, "user_id", userID, "error", err)
		return "", false, apperror.NewInternal()
	}

	var teamID string
	err = q.QueryRow(ctx, `
		SELECT ctp.team_id
		FROM contest_team_participants ctp
		JOIN team_members tm ON tm.team_id = ctp.team_id AND tm.user_id = $2
		WHERE ctp.contest_id = $1
		  AND ($2::uuid = ANY(ctp.selected_members) OR array_length(ctp.selected_members, 1) IS NULL)
		LIMIT 1`,
		contestID, userID,
	).Scan(&teamID)
	if err == nil {
		return teamID, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		slog.ErrorContext(ctx, "contest: failed to check team participation for standing",
			"contest_id", contestID, "user_id", userID, "error", err)
		return "", false, apperror.NewInternal()
	}

	return "", false, nil
}
