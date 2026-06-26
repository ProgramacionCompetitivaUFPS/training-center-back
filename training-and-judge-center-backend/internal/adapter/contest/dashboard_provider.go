package contest

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// DashboardProvider implements application/user.DashboardContestProvider.
type DashboardProvider struct {
	db infraPostgres.Querier
}

func NewDashboardProvider(db infraPostgres.Querier) *DashboardProvider {
	return &DashboardProvider{db: db}
}

// participationFilter returns the SQL fragment that checks whether the user
// is registered to a contest either individually or as part of a team.
const participationFilter = `(
	EXISTS (SELECT 1 FROM contest_registrations    cr  WHERE cr.contest_id  = c.id AND cr.user_id = $1)
	OR
	EXISTS (SELECT 1 FROM contest_team_participants ctp WHERE ctp.contest_id = c.id AND $1 = ANY(ctp.selected_members))
)`

func (p *DashboardProvider) GetUpcomingContests(ctx context.Context, userID string, limit int) ([]appuser.DashboardContest, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	rows, err := q.Query(ctx, `
		SELECT c.id,
		       c.name,
		       c.start_time,
		       EXTRACT(EPOCH FROM (c.end_time - c.start_time))::INTEGER / 60 AS duration_minutes
		FROM contests c
		WHERE c.start_time > NOW()
		  AND `+participationFilter+`
		ORDER BY c.start_time ASC
		LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		slog.ErrorContext(ctx, "dashboard: failed to query upcoming contests", "user_id", userID, "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()
	return scanContests(ctx, rows)
}

func (p *DashboardProvider) GetActiveContests(ctx context.Context, userID string, limit int) ([]appuser.DashboardContest, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	rows, err := q.Query(ctx, `
		SELECT c.id,
		       c.name,
		       c.start_time,
		       EXTRACT(EPOCH FROM (c.end_time - c.start_time))::INTEGER / 60 AS duration_minutes
		FROM contests c
		WHERE c.start_time <= NOW()
		  AND c.end_time   >= NOW()
		  AND `+participationFilter+`
		ORDER BY c.start_time ASC
		LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		slog.ErrorContext(ctx, "dashboard: failed to query active contests", "user_id", userID, "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()
	return scanContests(ctx, rows)
}

// GetFinishedContestResults returns up to limit finished contests with the
// user's ICPC score (problems solved + penalty) and their position.
//
// Position is computed as the count of distinct participants with strictly
// more problems solved than the user, plus one. Penalty tiebreaking is
// intentionally omitted for simplicity.
func (p *DashboardProvider) GetFinishedContestResults(ctx context.Context, userID string, limit int) ([]appuser.DashboardContestResult, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	rows, err := q.Query(ctx, `
		WITH finished AS (
			-- Last N finished contests the user participated in
			SELECT c.id, c.name, c.start_time, c.end_time, c.penalty AS contest_penalty
			FROM contests c
			WHERE c.end_time < NOW()
			  AND `+participationFilter+`
			ORDER BY c.end_time DESC
			LIMIT $2
		),
		user_first_ac AS (
			-- First accepted submission per problem per finished contest
			SELECT f.id AS contest_id,
			       s.problem_id,
			       MIN(s.submitted_at) AS first_ac_at
			FROM finished f
			JOIN submissions s ON s.contest_id = f.id
			                  AND s.user_id    = $1
			                  AND s.status     = 'ACCEPTED'
			GROUP BY f.id, s.problem_id
		),
		wrong_before AS (
			-- Wrong submissions before first AC per problem
			SELECT ufa.contest_id,
			       ufa.problem_id,
			       COUNT(s.id) AS cnt
			FROM user_first_ac ufa
			LEFT JOIN submissions s ON s.contest_id  = ufa.contest_id
			                       AND s.user_id     = $1
			                       AND s.problem_id  = ufa.problem_id
			                       AND s.status     != 'ACCEPTED'
			                       AND s.submitted_at < ufa.first_ac_at
			GROUP BY ufa.contest_id, ufa.problem_id
		),
		user_score AS (
			-- ICPC score per contest
			SELECT f.id AS contest_id,
			       f.name,
			       f.end_time,
			       COUNT(ufa.problem_id)::INTEGER AS problems_solved,
			       COALESCE(SUM(
			           EXTRACT(EPOCH FROM (ufa.first_ac_at - f.start_time)) / 60
			           + COALESCE(wb.cnt, 0) * f.contest_penalty
			       ), 0)::INTEGER AS penalty
			FROM finished f
			LEFT JOIN user_first_ac ufa ON ufa.contest_id = f.id
			LEFT JOIN wrong_before  wb  ON wb.contest_id  = ufa.contest_id
			                           AND wb.problem_id   = ufa.problem_id
			GROUP BY f.id, f.name, f.end_time, f.contest_penalty
		),
		positions AS (
			-- Count participants with strictly more problems solved (simplified ranking)
			SELECT us.contest_id,
			       COUNT(DISTINCT s2.user_id)::INTEGER + 1 AS position
			FROM user_score us
			LEFT JOIN submissions s2 ON s2.contest_id = us.contest_id
			                        AND s2.status      = 'ACCEPTED'
			                        AND s2.user_id    != $1
			GROUP BY us.contest_id, us.problems_solved
			HAVING COUNT(DISTINCT s2.problem_id) > us.problems_solved
			   OR  us.problems_solved = 0
		)
		SELECT us.contest_id,
		       us.name,
		       us.problems_solved,
		       us.penalty,
		       COALESCE(pos.position, 1) AS position
		FROM user_score us
		LEFT JOIN positions pos ON pos.contest_id = us.contest_id
		ORDER BY us.end_time DESC`,
		userID, limit,
	)
	if err != nil {
		slog.ErrorContext(ctx, "dashboard: failed to query finished contest results", "user_id", userID, "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	var results []appuser.DashboardContestResult
	for rows.Next() {
		var r appuser.DashboardContestResult
		if err := rows.Scan(&r.ContestID, &r.ContestName, &r.ProblemsSolved, &r.Penalty, &r.Position); err != nil {
			slog.ErrorContext(ctx, "dashboard: failed to scan contest result row", "error", err)
			return nil, apperror.NewInternal()
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "dashboard: contest results rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return results, nil
}

type contestRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}

func scanContests(ctx context.Context, rows contestRows) ([]appuser.DashboardContest, error) {
	var result []appuser.DashboardContest
	for rows.Next() {
		var c appuser.DashboardContest
		if err := rows.Scan(&c.ID, &c.Name, &c.StartTime, &c.DurationMinutes); err != nil {
			slog.ErrorContext(ctx, "dashboard: failed to scan contest row", "error", err)
			return nil, apperror.NewInternal()
		}
		result = append(result, c)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "dashboard: contest rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}
