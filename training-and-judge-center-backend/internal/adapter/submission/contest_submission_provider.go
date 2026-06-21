package submission

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
	domaincontest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ContestSubmissionProvider struct {
	db infraPostgres.Querier
}

func NewContestSubmissionProvider(db infraPostgres.Querier) *ContestSubmissionProvider {
	return &ContestSubmissionProvider{db: db}
}

func (p *ContestSubmissionProvider) GetContestForSubmission(ctx context.Context, groupID, contestID string) (*appsubmission.ContestSubmissionInfo, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	var id, name, gID string
	var startTime, endTime time.Time
	var enablePostContest bool

	err := q.QueryRow(ctx, `
		SELECT id, name, group_id, start_time, end_time, enable_post_contest
		FROM contests
		WHERE id = $1 AND group_id = $2
	`, contestID, groupID).Scan(&id, &name, &gID, &startTime, &endTime, &enablePostContest)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound(domaincontest.ErrCodeContestNotFound, "contest not found")
		}
		slog.ErrorContext(ctx, "submission: failed to get contest for submission",
			"contest_id", contestID, "group_id", groupID, "error", err)
		return nil, apperror.NewInternal()
	}

	rows, err := q.Query(ctx,
		`SELECT problem_id FROM contest_problems WHERE contest_id = $1`,
		contestID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "submission: failed to load contest problems",
			"contest_id", contestID, "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	var problemIDs []string
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			slog.ErrorContext(ctx, "submission: failed to scan contest problem row",
				"contest_id", contestID, "error", err)
			return nil, apperror.NewInternal()
		}
		problemIDs = append(problemIDs, pid)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "submission: contest problems rows error",
			"contest_id", contestID, "error", err)
		return nil, apperror.NewInternal()
	}

	return &appsubmission.ContestSubmissionInfo{
		ID:                id,
		Name:              name,
		GroupID:           gID,
		StartTime:         startTime,
		EndTime:           endTime,
		EnablePostContest: enablePostContest,
		ProblemIDs:        problemIDs,
	}, nil
}
