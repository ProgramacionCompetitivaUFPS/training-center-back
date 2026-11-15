package submission

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
	domaincontest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ContestProvider struct {
	db infraPostgres.Querier
}

func NewContestProvider(db infraPostgres.Querier) *ContestProvider {
	return &ContestProvider{db: db}
}

func (p *ContestProvider) GetContestByID(ctx context.Context, contestID string) (*appsubmission.ContestDisplay, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	var id, name, groupID string
	err := q.QueryRow(ctx, `SELECT id, name, group_id FROM contests WHERE id = $1`, contestID).Scan(&id, &name, &groupID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound(domaincontest.ErrCodeContestNotFound, "contest not found")
		}
		slog.ErrorContext(ctx, "submission: failed to get contest display", "contest_id", contestID, "error", err)
		return nil, apperror.NewInternal()
	}

	return &appsubmission.ContestDisplay{ID: id, Name: name, GroupID: groupID}, nil
}

func (p *ContestProvider) GetProblemPosition(ctx context.Context, contestID, problemID string) (*int, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	var position int
	err := q.QueryRow(ctx, `SELECT "order" FROM contest_problems WHERE contest_id = $1 AND problem_id = $2`, contestID, problemID).Scan(&position)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "submission: failed to get problem position in contest", "contest_id", contestID, "problem_id", problemID, "error", err)
		return nil, apperror.NewInternal()
	}
	return &position, nil
}

func (p *ContestProvider) GetContestsByIDs(ctx context.Context, contestIDs []string) (map[string]*appsubmission.ContestDisplay, error) {
	if len(contestIDs) == 0 {
		return map[string]*appsubmission.ContestDisplay{}, nil
	}
	q := infraPostgres.GetQuerier(ctx, p.db)

	rows, err := q.Query(ctx, `SELECT id, name, group_id FROM contests WHERE id = ANY($1)`, contestIDs)
	if err != nil {
		slog.ErrorContext(ctx, "submission: failed to get contest displays", "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	result := make(map[string]*appsubmission.ContestDisplay, len(contestIDs))
	for rows.Next() {
		var id, name, groupID string
		if err := rows.Scan(&id, &name, &groupID); err != nil {
			slog.ErrorContext(ctx, "submission: failed to scan contest display row", "error", err)
			return nil, apperror.NewInternal()
		}
		result[id] = &appsubmission.ContestDisplay{ID: id, Name: name, GroupID: groupID}
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "submission: error iterating contest display rows", "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}
