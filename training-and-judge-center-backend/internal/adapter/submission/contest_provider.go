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

	var id, name string
	err := q.QueryRow(ctx, `SELECT id, name FROM contests WHERE id = $1`, contestID).Scan(&id, &name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound(domaincontest.ErrCodeContestNotFound, "contest not found")
		}
		slog.ErrorContext(ctx, "submission: failed to get contest display", "contest_id", contestID, "error", err)
		return nil, apperror.NewInternal()
	}

	return &appsubmission.ContestDisplay{ID: id, Name: name}, nil
}
