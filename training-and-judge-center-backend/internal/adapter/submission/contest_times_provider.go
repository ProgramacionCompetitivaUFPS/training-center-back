package submission

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ContestTimesProvider implements appsubmission.ContestTimesProvider.
type ContestTimesProvider struct {
	db infraPostgres.Querier
}

func NewContestTimesProvider(db infraPostgres.Querier) *ContestTimesProvider {
	return &ContestTimesProvider{db: db}
}

func (p *ContestTimesProvider) GetContestTimes(ctx context.Context, contestID string) (startTime, endTime time.Time, err error) {
	q := infraPostgres.GetQuerier(ctx, p.db)
	err = q.QueryRow(ctx,
		`SELECT start_time, end_time FROM contests WHERE id = $1`, contestID,
	).Scan(&startTime, &endTime)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, time.Time{}, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
		}
		slog.ErrorContext(ctx, "submission: failed to get contest times", "contest_id", contestID, "error", err)
		return time.Time{}, time.Time{}, apperror.NewInternal()
	}
	return startTime, endTime, nil
}
