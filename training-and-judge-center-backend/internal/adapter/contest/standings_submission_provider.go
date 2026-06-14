package contest

import (
	"context"
	"log/slog"
	"time"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appContest "github.com/training-judge-center/backend/internal/application/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// StandingsSubmissionProvider implements application/contest.StandingsSubmissionProvider
// by querying submissions directly from Postgres.
type StandingsSubmissionProvider struct {
	db infraPostgres.Querier
}

func NewStandingsSubmissionProvider(db infraPostgres.Querier) *StandingsSubmissionProvider {
	return &StandingsSubmissionProvider{db: db}
}

func (p *StandingsSubmissionProvider) ListByContest(ctx context.Context, contestID string) ([]appContest.ContestSubmissionData, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	rows, err := q.Query(ctx, `
		SELECT user_id, problem_id, status, submitted_at
		FROM submissions
		WHERE contest_id = $1
		  AND status NOT IN ('PENDING', 'RUNNING')
		ORDER BY submitted_at ASC`,
		contestID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list contest submissions for standings", "contest_id", contestID, "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	var result []appContest.ContestSubmissionData
	for rows.Next() {
		var userID, problemID, status string
		var submittedAt time.Time
		if err := rows.Scan(&userID, &problemID, &status, &submittedAt); err != nil {
			slog.ErrorContext(ctx, "failed to scan standings submission row", "error", err)
			return nil, apperror.NewInternal()
		}
		result = append(result, appContest.ContestSubmissionData{
			UserID:      userID,
			ProblemID:   problemID,
			Status:      status,
			SubmittedAt: submittedAt,
		})
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "standings submissions rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}
