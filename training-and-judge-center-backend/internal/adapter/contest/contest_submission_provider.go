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
		// individual registrant OR selected team member with given nickname
		where += fmt.Sprintf(`
			AND (
				(s.standing_id::text = s.user_id::text AND u.nickname = $%d)
				OR EXISTS (
					SELECT 1 FROM contest_team_participants ctp2
					JOIN team_members tm ON tm.team_id = ctp2.team_id
					JOIN users u2 ON u2.id = tm.user_id AND u2.nickname = $%d
					WHERE ctp2.contest_id = s.contest_id AND ctp2.team_id::text = s.standing_id::text
					  AND (u2.id = ANY(ctp2.selected_members) OR array_length(ctp2.selected_members, 1) IS NULL)
				)
			)`, n, n)
		args = append(args, *filters.Nickname)
		n++
	}

	// correlated subquery avoids multiple rows per submission from team joins
	query := fmt.Sprintf(`
		SELECT
			s.id,
			s.problem_id,
			p.slug,
			p.title,
			COALESCE(cp."order", 0),
			s.user_id,
			s.standing_id,
			u.nickname,
			u.name,
			t.id,
			t.name,
			COALESCE(
				(
					SELECT ARRAY_AGG(u2.nickname ORDER BY u2.nickname)
					FROM unnest(COALESCE(ctp.selected_members, ARRAY[]::uuid[])) AS mid
					JOIN users u2 ON u2.id = mid
				),
				ARRAY[]::text[]
			),
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
		LEFT JOIN contest_team_participants ctp ON ctp.team_id::text = s.standing_id::text AND ctp.contest_id = s.contest_id
		LEFT JOIN teams t ON t.id = ctp.team_id
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
		var (
			d               appContest.RichSubmissionData
			standingID      *string
			judgedAt        *time.Time
			timeMs, memKb   *int
			teamID          *string
			teamName        *string
			memberNicknames []string
		)

		if err := rows.Scan(
			&d.ID,
			&d.ProblemID,
			&d.ProblemSlug,
			&d.ProblemTitle,
			&d.ProblemOrder,
			&d.UserID,
			&standingID,
			&d.Nickname,
			&d.SubmitterName,
			&teamID,
			&teamName,
			&memberNicknames,
			&d.Language,
			&d.Status,
			&d.SubmittedAt,
			&judgedAt,
			&timeMs,
			&memKb,
		); err != nil {
			slog.ErrorContext(ctx, "failed to scan contest submission row", "error", err)
			return nil, apperror.NewInternal()
		}
		if standingID != nil {
			d.StandingID = *standingID
		}
		d.JudgedAt = judgedAt
		d.TimeMs = timeMs
		d.MemoryKb = memKb
		d.TeamID = teamID
		d.TeamName = teamName
		d.TeamMemberNicknames = memberNicknames
		result = append(result, d)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "contest submissions rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}
