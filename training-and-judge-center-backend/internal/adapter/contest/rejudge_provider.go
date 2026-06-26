package contest

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ContestRejudgeProvider implements appProblem.ContestRejudgeProvider.
type ContestRejudgeProvider struct {
	db infraPostgres.Querier
}

func NewContestRejudgeProvider(db infraPostgres.Querier) *ContestRejudgeProvider {
	return &ContestRejudgeProvider{db: db}
}

var _ appProblem.ContestRejudgeProvider = (*ContestRejudgeProvider)(nil)

func (p *ContestRejudgeProvider) GetContestForRejudge(ctx context.Context, contestID string) (*appProblem.ContestRejudgeInfo, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	var info appProblem.ContestRejudgeInfo
	var startTime, endTime time.Time
	err := q.QueryRow(ctx, `
		SELECT id, owner_id, group_id, start_time, end_time
		FROM contests WHERE id = $1`,
		contestID,
	).Scan(&info.ID, &info.OwnerID, &info.GroupID, &startTime, &endTime)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
		}
		slog.ErrorContext(ctx, "rejudge: failed to get contest", "contest_id", contestID, "error", err)
		return nil, apperror.NewInternal()
	}
	info.StartTime = startTime
	info.EndTime = endTime
	return &info, nil
}

func (p *ContestRejudgeProvider) IsProblemInContest(ctx context.Context, contestID, problemID string) (bool, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)
	var exists bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM contest_problems
			WHERE contest_id = $1 AND problem_id = $2
		)`, contestID, problemID,
	).Scan(&exists)
	if err != nil {
		slog.ErrorContext(ctx, "rejudge: failed to check problem in contest", "contest_id", contestID, "problem_id", problemID, "error", err)
		return false, apperror.NewInternal()
	}
	return exists, nil
}

func (p *ContestRejudgeProvider) IsLeadOfGroup(ctx context.Context, userID, groupID string) (bool, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)
	var exists bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM group_members
			WHERE group_id = $1 AND user_id = $2 AND member_role = 'LEAD'
		)`, groupID, userID,
	).Scan(&exists)
	if err != nil {
		slog.ErrorContext(ctx, "rejudge: failed to check lead membership", "group_id", groupID, "user_id", userID, "error", err)
		return false, apperror.NewInternal()
	}
	return exists, nil
}
