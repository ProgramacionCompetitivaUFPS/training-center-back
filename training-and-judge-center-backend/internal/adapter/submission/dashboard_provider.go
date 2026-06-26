package submission

import (
	"context"
	"log/slog"
	"time"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// DashboardProvider implements application/user.DashboardSubmissionProvider.
type DashboardProvider struct {
	db infraPostgres.Querier
}

func NewDashboardProvider(db infraPostgres.Querier) *DashboardProvider {
	return &DashboardProvider{db: db}
}

func (p *DashboardProvider) GetRecentSubmissions(ctx context.Context, userID string, limit int) ([]appuser.DashboardSubmission, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	rows, err := q.Query(ctx, `
		SELECT id,
		       COALESCE(problem_slug, '')  AS problem_slug,
		       COALESCE(problem_title, '') AS problem_title,
		       status,
		       language,
		       submitted_at,
		       time_ms,
		       memory_kb
		FROM submissions
		WHERE user_id = $1
		ORDER BY submitted_at DESC
		LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		slog.ErrorContext(ctx, "dashboard: failed to query recent submissions", "user_id", userID, "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	var result []appuser.DashboardSubmission
	for rows.Next() {
		var s appuser.DashboardSubmission
		if err := rows.Scan(
			&s.ID, &s.ProblemSlug, &s.ProblemTitle,
			&s.Verdict, &s.Language, &s.SubmittedAt,
			&s.ExecutionTime, &s.MemoryKb,
		); err != nil {
			slog.ErrorContext(ctx, "dashboard: failed to scan submission row", "error", err)
			return nil, apperror.NewInternal()
		}
		result = append(result, s)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "dashboard: submissions rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}

func (p *DashboardProvider) GetSubmissionDates(ctx context.Context, userID string) ([]time.Time, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	rows, err := q.Query(ctx, `
		SELECT DISTINCT (submitted_at AT TIME ZONE 'UTC')::DATE AS submission_date
		FROM submissions
		WHERE user_id = $1
		ORDER BY submission_date ASC`,
		userID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "dashboard: failed to query submission dates", "user_id", userID, "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	var dates []time.Time
	for rows.Next() {
		var d time.Time
		if err := rows.Scan(&d); err != nil {
			slog.ErrorContext(ctx, "dashboard: failed to scan submission date", "error", err)
			return nil, apperror.NewInternal()
		}
		dates = append(dates, d.UTC())
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "dashboard: submission dates rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return dates, nil
}
