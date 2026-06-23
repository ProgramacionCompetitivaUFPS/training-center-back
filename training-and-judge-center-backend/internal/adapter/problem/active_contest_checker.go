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
