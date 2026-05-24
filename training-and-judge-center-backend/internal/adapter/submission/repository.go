package submission

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	infrapostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	domainsubmission "github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type Repository struct {
	db infrapostgres.Querier
}

func NewRepository(db infrapostgres.Querier) *Repository {
	return &Repository{db: db}
}

const submissionColumns = `id, problem_id, user_id, contest_id, language, status,
	source_code_path, submitted_at, judged_at, time_ms, memory_kb, compile_log`

func (r *Repository) Save(ctx context.Context, s *domainsubmission.Submission) error {
	q := infrapostgres.GetQuerier(ctx, r.db)
	_, err := q.Exec(ctx, `
		INSERT INTO submissions
			(id, problem_id, user_id, contest_id, language, status,
			 source_code_path, submitted_at, judged_at, time_ms, memory_kb, compile_log)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		s.ID(),
		s.ProblemID(),
		s.UserID().Value(),
		s.ContestID(),
		s.Language().String(),
		s.Status().String(),
		s.SourceCodePath(),
		s.SubmittedAt(),
		s.JudgedAt(),
		s.TimeMs(),
		s.MemoryKb(),
		s.CompileLog(),
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to save submission", "error", err, "submission_id", s.ID())
		return apperror.NewInternal()
	}
	return nil
}

func (r *Repository) FindByID(ctx context.Context, id domainsubmission.SubmissionID) (*domainsubmission.Submission, error) {
	q := infrapostgres.GetQuerier(ctx, r.db)

	var (
		sID, problemID, userIDStr, lang, status, sourceCodePath string
		contestID                                               *string
		submittedAt                                             time.Time
		judgedAt                                                *time.Time
		timeMs, memoryKb                                        *int
		compileLog                                              *string
	)
	err := q.QueryRow(ctx,
		`SELECT `+submissionColumns+` FROM submissions WHERE id=$1`, id,
	).Scan(
		&sID, &problemID, &userIDStr, &contestID,
		&lang, &status, &sourceCodePath,
		&submittedAt, &judgedAt, &timeMs, &memoryKb, &compileLog,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound(domainsubmission.ErrCodeSubmissionNotFound, "submission not found")
		}
		slog.ErrorContext(ctx, "failed to find submission", "error", err, "submission_id", id)
		return nil, apperror.NewInternal()
	}

	return domainsubmission.RestoreSubmission(
		sID,
		problemID,
		shared.RestoreUserID(userIDStr),
		contestID,
		domainsubmission.RestoreLanguage(lang),
		domainsubmission.RestoreStatus(status),
		sourceCodePath,
		submittedAt,
		judgedAt,
		timeMs,
		memoryKb,
		compileLog,
	), nil
}

func (r *Repository) GetByID(ctx context.Context, id domainsubmission.SubmissionID) (*domainsubmission.Submission, error) {
	return r.FindByID(ctx, id)
}

func (r *Repository) Update(ctx context.Context, s *domainsubmission.Submission) error {
	q := infrapostgres.GetQuerier(ctx, r.db)
	tag, err := q.Exec(ctx, `
		UPDATE submissions
		SET status=$2, judged_at=$3, time_ms=$4, memory_kb=$5, compile_log=$6
		WHERE id=$1`,
		s.ID(),
		s.Status().String(),
		s.JudgedAt(),
		s.TimeMs(),
		s.MemoryKb(),
		s.CompileLog(),
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to update submission", "error", err, "submission_id", s.ID())
		return apperror.NewInternal()
	}
	if tag.RowsAffected() == 0 {
		return apperror.NewNotFound(domainsubmission.ErrCodeSubmissionNotFound, "submission not found")
	}
	return nil
}

func (r *Repository) List(ctx context.Context, filters domainsubmission.ListFilters) ([]*domainsubmission.Submission, int, error) {
	q := infrapostgres.GetQuerier(ctx, r.db)

	args := []interface{}{}
	n := 1
	where := "WHERE 1=1"

	if filters.UserID != nil {
		where += fmt.Sprintf(" AND user_id=$%d", n)
		args = append(args, filters.UserID.Value())
		n++
	}
	if filters.ProblemID != nil {
		where += fmt.Sprintf(" AND problem_id=$%d", n)
		args = append(args, *filters.ProblemID)
		n++
	}
	if filters.ContestID != nil {
		where += fmt.Sprintf(" AND contest_id=$%d", n)
		args = append(args, *filters.ContestID)
		n++
	}

	var total int
	if err := q.QueryRow(ctx, `SELECT COUNT(*) FROM submissions `+where, args...).Scan(&total); err != nil {
		slog.ErrorContext(ctx, "failed to count submissions", "error", err)
		return nil, 0, apperror.NewInternal()
	}

	limitPos := n
	offsetPos := n + 1
	args = append(args, filters.Limit, (filters.Page-1)*filters.Limit)

	query := `SELECT ` + submissionColumns + ` FROM submissions ` + where +
		` ORDER BY submitted_at DESC` +
		` LIMIT $` + strconv.Itoa(limitPos) +
		` OFFSET $` + strconv.Itoa(offsetPos)

	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list submissions", "error", err)
		return nil, 0, apperror.NewInternal()
	}
	defer rows.Close()

	result := []*domainsubmission.Submission{}
	for rows.Next() {
		var (
			sID, problemID, userIDStr, lang, status, sourceCodePath string
			contestID                                               *string
			submittedAt                                             time.Time
			judgedAt                                                *time.Time
			timeMs, memoryKb                                        *int
			compileLog                                              *string
		)
		if err := rows.Scan(
			&sID, &problemID, &userIDStr, &contestID,
			&lang, &status, &sourceCodePath,
			&submittedAt, &judgedAt, &timeMs, &memoryKb, &compileLog,
		); err != nil {
			slog.ErrorContext(ctx, "failed to scan submission row", "error", err)
			return nil, 0, apperror.NewInternal()
		}
		result = append(result, domainsubmission.RestoreSubmission(
			sID,
			problemID,
			shared.RestoreUserID(userIDStr),
			contestID,
			domainsubmission.RestoreLanguage(lang),
			domainsubmission.RestoreStatus(status),
			sourceCodePath,
			submittedAt,
			judgedAt,
			timeMs,
			memoryKb,
			compileLog,
		))
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "list submissions rows error", "error", err)
		return nil, 0, apperror.NewInternal()
	}

	return result, total, nil
}
