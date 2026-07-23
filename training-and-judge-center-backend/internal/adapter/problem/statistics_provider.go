package problem

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type StatisticsProvider struct {
	db *pgxpool.Pool
}

func NewStatisticsProvider(db *pgxpool.Pool) *StatisticsProvider {
	return &StatisticsProvider{db: db}
}

func (p *StatisticsProvider) GetByProblemID(ctx context.Context, problemID string) (*appProblem.ProblemStats, error) {
	stats := &appProblem.ProblemStats{}

	if err := p.db.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(DISTINCT user_id),
			COUNT(DISTINCT CASE WHEN status = 'ACCEPTED' THEN user_id END)
		FROM submissions
		WHERE problem_id = $1`, problemID,
	).Scan(&stats.TotalSubmissions, &stats.UniqueAttempted, &stats.UniqueSolved); err != nil {
		slog.ErrorContext(ctx, "failed to query problem submission totals", "error", err, "problem_id", problemID)
		return nil, apperror.NewInternal()
	}

	rows, err := p.db.Query(ctx, `
		SELECT
			language,
			COUNT(DISTINCT user_id) AS users_attempted,
			COUNT(DISTINCT CASE WHEN status = 'ACCEPTED' THEN user_id END) AS users_accepted
		FROM submissions
		WHERE problem_id = $1
		GROUP BY language
		ORDER BY
			CAST(COUNT(DISTINCT CASE WHEN status = 'ACCEPTED' THEN user_id END) AS float8) /
			NULLIF(COUNT(DISTINCT user_id), 0) DESC NULLS LAST`, problemID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to query language stats", "error", err, "problem_id", problemID)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	for rows.Next() {
		var ls appProblem.LanguageStat
		if err := rows.Scan(&ls.Language, &ls.UsersAttempted, &ls.UsersAccepted); err != nil {
			slog.ErrorContext(ctx, "failed to scan language stat row", "error", err)
			return nil, apperror.NewInternal()
		}
		stats.ByLanguage = append(stats.ByLanguage, ls)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "language stats rows error", "error", err)
		return nil, apperror.NewInternal()
	}

	vrows, err := p.db.Query(ctx, `
		SELECT status, COUNT(*) AS cnt
		FROM submissions
		WHERE problem_id = $1
		GROUP BY status
		ORDER BY cnt DESC`, problemID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to query verdict distribution", "error", err, "problem_id", problemID)
		return nil, apperror.NewInternal()
	}
	defer vrows.Close()

	for vrows.Next() {
		var vs appProblem.VerdictStat
		if err := vrows.Scan(&vs.Verdict, &vs.Count); err != nil {
			slog.ErrorContext(ctx, "failed to scan verdict stat row", "error", err)
			return nil, apperror.NewInternal()
		}
		stats.ByVerdict = append(stats.ByVerdict, vs)
	}
	if err := vrows.Err(); err != nil {
		slog.ErrorContext(ctx, "verdict distribution rows error", "error", err)
		return nil, apperror.NewInternal()
	}

	return stats, nil
}
