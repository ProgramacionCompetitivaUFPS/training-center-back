package problem

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ContestAssociationRemover struct {
	db infraPostgres.Querier
}

func NewContestAssociationRemover(db infraPostgres.Querier) *ContestAssociationRemover {
	return &ContestAssociationRemover{db: db}
}

func (r *ContestAssociationRemover) RemoveProblemFromContests(ctx context.Context, problemID string) error {
	q := infraPostgres.GetQuerier(ctx, r.db)

	if _, err := q.Exec(ctx, `DELETE FROM contest_problems WHERE problem_id = $1::uuid`, problemID); err != nil {
		slog.ErrorContext(ctx, "failed to remove problem from contests", "problem_id", problemID, "error", err)
		return apperror.NewInternal()
	}
	return nil
}
