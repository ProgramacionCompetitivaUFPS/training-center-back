package problem

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// Positions are pushed out of range first because (contest_id, "order") is unique and cannot be shifted row by row.
const (
	shiftOrdersQuery = `UPDATE contest_problems SET "order" = "order" + 1000000 WHERE contest_id = ANY($1::uuid[])`

	renumberOrdersQuery = `
		UPDATE contest_problems cp SET "order" = ranked.position
		FROM (
			SELECT id, ROW_NUMBER() OVER (PARTITION BY contest_id ORDER BY "order") AS position
			FROM contest_problems
			WHERE contest_id = ANY($1::uuid[])
		) ranked
		WHERE cp.id = ranked.id`
)

type ContestAssociationRemover struct {
	db infraPostgres.Querier
}

func NewContestAssociationRemover(db infraPostgres.Querier) *ContestAssociationRemover {
	return &ContestAssociationRemover{db: db}
}

func (r *ContestAssociationRemover) RemoveProblemFromContests(ctx context.Context, problemID string) error {
	q := infraPostgres.GetQuerier(ctx, r.db)

	contestIDs, err := r.deleteAssociations(ctx, q, problemID)
	if err != nil {
		return err
	}
	if len(contestIDs) == 0 {
		return nil
	}

	for _, query := range []string{shiftOrdersQuery, renumberOrdersQuery} {
		if _, err := q.Exec(ctx, query, contestIDs); err != nil {
			slog.ErrorContext(ctx, "failed to renumber contest problems", "problem_id", problemID, "error", err)
			return apperror.NewInternal()
		}
	}
	return nil
}

func (r *ContestAssociationRemover) deleteAssociations(ctx context.Context, q infraPostgres.Querier, problemID string) ([]string, error) {
	rows, err := q.Query(ctx, `DELETE FROM contest_problems WHERE problem_id = $1::uuid RETURNING contest_id`, problemID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to remove problem from contests", "problem_id", problemID, "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	var contestIDs []string
	for rows.Next() {
		var contestID string
		if err := rows.Scan(&contestID); err != nil {
			slog.ErrorContext(ctx, "failed to scan contest of removed problem", "problem_id", problemID, "error", err)
			return nil, apperror.NewInternal()
		}
		contestIDs = append(contestIDs, contestID)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "failed to read contests of removed problem", "problem_id", problemID, "error", err)
		return nil, apperror.NewInternal()
	}
	return contestIDs, nil
}
