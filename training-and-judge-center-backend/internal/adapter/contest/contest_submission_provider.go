package contest

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appContest "github.com/training-judge-center/backend/internal/application/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ContestSubmissionProvider implements application/contest.ContestSubmissionProvider
// by querying submissions with problem and user details from Postgres.
type ContestSubmissionProvider struct {
	db infraPostgres.Querier
}

func NewContestSubmissionProvider(db infraPostgres.Querier) *ContestSubmissionProvider {
	return &ContestSubmissionProvider{db: db}
}

func (p *ContestSubmissionProvider) ListByContest(
	ctx context.Context,
	contestID string,
	filters appContest.ContestSubmissionFilters,
) ([]appContest.RichSubmissionData, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	args := []any{contestID}
	where := ""
	n := 2

	if filters.ProblemSlug != nil {
		where += fmt.Sprintf(" AND p.slug = $%d", n)
		args = append(args, *filters.ProblemSlug)
		n++
	}
	if filters.Nickname != nil {
		where += fmt.Sprintf(" AND u.nickname = $%d", n)
		args = append(args, *filters.Nickname)
		n++
	}

	query := fmt.Sprintf(`
		SELECT
			s.id,
			s.problem_id,
			p.slug,
			p.title,
			COALESCE(cp."order", 0),
			s.user_id,
			u.nickname,
			s.language,
			s.status,
			s.submitted_at,
			s.judged_at,
			s.time_ms,
			s.memory_kb
		FROM submissions s
		JOIN problems p ON p.id = s.problem_id
		JOIN users u ON u.id = s.user_id
		LEFT JOIN contest_problems cp ON cp.problem_id = s.problem_id AND cp.contest_id = s.contest_id
		WHERE s.contest_id = $1%s
		ORDER BY s.submitted_at DESC, s.id`, where)

	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list contest submissions", "contest_id", contestID, "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	var result []appContest.RichSubmissionData
	for rows.Next() {
		var d appContest.RichSubmissionData
		var judgedAt *time.Time
		var timeMs, memoryKb *int

		if err := rows.Scan(
			&d.ID,
			&d.ProblemID,
			&d.ProblemSlug,
			&d.ProblemTitle,
			&d.ProblemOrder,
			&d.UserID,
			&d.Nickname,
			&d.Language,
			&d.Status,
			&d.SubmittedAt,
			&judgedAt,
			&timeMs,
			&memoryKb,
		); err != nil {
			slog.ErrorContext(ctx, "failed to scan contest submission row", "error", err)
			return nil, apperror.NewInternal()
		}
		d.JudgedAt = judgedAt
		d.TimeMs = timeMs
		d.MemoryKb = memoryKb
		result = append(result, d)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "contest submissions rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}
