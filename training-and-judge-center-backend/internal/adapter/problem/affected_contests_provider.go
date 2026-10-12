package problem

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var _ appProblem.AffectedContestsProvider = (*AffectedContestsProvider)(nil)

type AffectedContestsProvider struct {
	db infraPostgres.Querier
}

func NewAffectedContestsProvider(db infraPostgres.Querier) *AffectedContestsProvider {
	return &AffectedContestsProvider{db: db}
}

func (p *AffectedContestsProvider) ListOpenContestsOfProblem(ctx context.Context, problemID string) ([]appProblem.ContestRef, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	rows, err := q.Query(ctx, `
		SELECT c.id, c.name, c.group_id, c.start_time, c.end_time
		FROM contest_problems cp
		JOIN contests c ON c.id = cp.contest_id
		WHERE cp.problem_id = $1::uuid
		  AND c.end_time >= NOW()
		ORDER BY c.start_time`, problemID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list open contests of problem", "problem_id", problemID, "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	var contests []appProblem.ContestRef
	for rows.Next() {
		var c appProblem.ContestRef
		if err := rows.Scan(&c.ID, &c.Name, &c.GroupID, &c.StartTime, &c.EndTime); err != nil {
			slog.ErrorContext(ctx, "failed to scan open contest of problem", "problem_id", problemID, "error", err)
			return nil, apperror.NewInternal()
		}
		contests = append(contests, c)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "failed to read open contests of problem", "problem_id", problemID, "error", err)
		return nil, apperror.NewInternal()
	}
	return contests, nil
}
