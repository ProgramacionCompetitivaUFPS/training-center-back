package problem

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ActiveContestChecker struct {
	db infraPostgres.Querier
}

func NewActiveContestChecker(db infraPostgres.Querier) *ActiveContestChecker {
	return &ActiveContestChecker{db: db}
}

func (c *ActiveContestChecker) IsProblemInActiveContest(ctx context.Context, problemID string) (bool, error) {
	q := infraPostgres.GetQuerier(ctx, c.db)

	var exists bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM contest_problems cp
			JOIN contests co ON co.id = cp.contest_id
			WHERE cp.problem_id = $1::uuid
			  AND co.start_time <= NOW()
			  AND co.end_time >= NOW()
		)`, problemID,
	).Scan(&exists)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check if problem is in active contest", "problem_id", problemID, "error", err)
		return false, apperror.NewInternal()
	}
	return exists, nil
}

func (c *ActiveContestChecker) HasContestSubmissions(ctx context.Context, problemID string) (bool, error) {
	q := infraPostgres.GetQuerier(ctx, c.db)

	// A submission insert holds FOR KEY SHARE on the problem row (its foreign key), which conflicts
	// with FOR UPDATE: this waits for an insert in flight, and the next statement, a new snapshot
	// in READ COMMITTED, already sees it. Outside a transaction the lock ends with the statement.
	if _, err := q.Exec(ctx, `SELECT 1 FROM problems WHERE id = $1::uuid FOR UPDATE`, problemID); err != nil {
		slog.ErrorContext(ctx, "failed to lock problem to check its contest submissions", "problem_id", problemID, "error", err)
		return false, apperror.NewInternal()
	}

	var exists bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM submissions
			WHERE problem_id = $1::uuid AND contest_id IS NOT NULL
		)`, problemID,
	).Scan(&exists)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check contest submissions of problem", "problem_id", problemID, "error", err)
		return false, apperror.NewInternal()
	}
	return exists, nil
}
