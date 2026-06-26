package submission

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/errgroup"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	domainSubmission "github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type Repository struct {
	db infraPostgres.Querier
}

func NewRepository(db infraPostgres.Querier) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Save(ctx context.Context, s *domainSubmission.Submission) error {
	q := infraPostgres.GetQuerier(ctx, r.db)
	_, err := q.Exec(ctx, `
		INSERT INTO submissions (
			id, problem_id, user_id, contest_id, standing_id,
			language, compiler, status, visibility, source_code_path,
			file_hash, file_size, submitted_at,
			problem_title, problem_slug
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		ON CONFLICT (id) DO UPDATE SET
			status        = EXCLUDED.status,
			visibility    = EXCLUDED.visibility,
			compiler      = EXCLUDED.compiler,
			file_hash     = EXCLUDED.file_hash,
			file_size     = EXCLUDED.file_size,
			problem_title = EXCLUDED.problem_title,
			problem_slug  = EXCLUDED.problem_slug
	`,
		s.ID(),
		nilIfEmpty(s.ProblemID()),
		s.UserID().String(),
		s.ContestID(),
		s.StandingID(),
		s.Language().String(),
		s.Compiler(),
		s.Status().String(),
		s.Visibility().String(),
		s.SourceCodePath(),
		nilIfEmpty(s.FileHash()),
		nilIfZero(s.FileSize()),
		s.SubmittedAt(),
		nilIfEmpty(s.ProblemTitle()),
		nilIfEmpty(s.ProblemSlug()),
	)
	if err != nil {
		slog.ErrorContext(ctx, "submission: failed to save", "id", s.ID(), "error", err)
		return apperror.NewInternal()
	}
	return nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (*domainSubmission.Submission, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)
	sub, err := scanRow(ctx, q.QueryRow(ctx, selectCols+` WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound(domainSubmission.ErrCodeSubmissionNotFound, "submission not found")
		}
		return nil, err
	}
	return sub, nil
}

func (r *Repository) List(ctx context.Context, f domainSubmission.ListFilters) ([]*domainSubmission.Submission, int, error) {
	where, args := buildWhere(f)

	li, oi := len(args)+1, len(args)+2
	dataArgs := append(append([]any(nil), args...), f.Limit, (f.Page-1)*f.Limit)
	dataQuery := selectCols + where + fmt.Sprintf(" ORDER BY submitted_at DESC LIMIT $%d OFFSET $%d", li, oi)

	// Obtain the querier before spawning goroutines so both share the same tx if active.
	q := infraPostgres.GetQuerier(ctx, r.db)
	g, gCtx := errgroup.WithContext(ctx)

	var total int
	var result []*domainSubmission.Submission

	g.Go(func() error {
		if err := q.QueryRow(gCtx, "SELECT COUNT(*) FROM submissions"+where, args...).Scan(&total); err != nil {
			slog.ErrorContext(gCtx, "submission: failed to count", "error", err)
			return apperror.NewInternal()
		}
		return nil
	})

	g.Go(func() error {
		rows, err := q.Query(gCtx, dataQuery, dataArgs...)
		if err != nil {
			slog.ErrorContext(gCtx, "submission: failed to list", "error", err)
			return apperror.NewInternal()
		}
		defer rows.Close()

		for rows.Next() {
			sub, err := scanRow(gCtx, rows)
			if err != nil {
				return err
			}
			result = append(result, sub)
		}
		if rows.Err() != nil {
			slog.ErrorContext(gCtx, "submission: error iterating rows", "error", rows.Err())
			return apperror.NewInternal()
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, 0, err
	}
	if result == nil {
		result = []*domainSubmission.Submission{}
	}
	return result, total, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*domainSubmission.Submission, error) {
	return r.FindByID(ctx, id)
}

func (r *Repository) Update(ctx context.Context, s *domainSubmission.Submission) error {
	q := infraPostgres.GetQuerier(ctx, r.db)
	tag, err := q.Exec(ctx, `
		UPDATE submissions SET
			status      = $2,
			judged_at   = $3,
			time_ms     = $4,
			memory_kb   = $5,
			compile_log = $6
		WHERE id = $1
	`,
		s.ID(),
		s.Status().String(),
		s.JudgedAt(),
		s.TimeMs(),
		s.MemoryKb(),
		s.CompileLog(),
	)
	if err != nil {
		slog.ErrorContext(ctx, "submission: failed to update", "id", s.ID(), "error", err)
		return apperror.NewInternal()
	}
	if tag.RowsAffected() == 0 {
		return apperror.NewNotFound(domainSubmission.ErrCodeSubmissionNotFound, "submission not found")
	}
	return nil
}

func (r *Repository) FindLastByUserAndProblem(ctx context.Context, userID, problemID string, contestID *string) (*domainSubmission.Submission, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)
	var sub *domainSubmission.Submission
	var err error
	if contestID == nil {
		sub, err = scanRow(ctx, q.QueryRow(ctx,
			selectCols+` WHERE user_id = $1 AND problem_id = $2 AND contest_id IS NULL ORDER BY submitted_at DESC LIMIT 1`,
			userID, problemID))
	} else {
		sub, err = scanRow(ctx, q.QueryRow(ctx,
			selectCols+` WHERE user_id = $1 AND problem_id = $2 AND contest_id = $3 ORDER BY submitted_at DESC LIMIT 1`,
			userID, problemID, *contestID))
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound(domainSubmission.ErrCodeSubmissionNotFound, "no previous submission")
		}
		return nil, err
	}
	return sub, nil
}

func (r *Repository) ExistsByHashAndUserAndProblem(ctx context.Context, fileHash, userID, problemID string, contestID *string) (bool, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)
	var query string
	var args []any
	if contestID == nil {
		query = `SELECT EXISTS(SELECT 1 FROM submissions WHERE file_hash = $1 AND user_id = $2 AND problem_id = $3 AND contest_id IS NULL)`
		args = []any{fileHash, userID, problemID}
	} else {
		query = `SELECT EXISTS(SELECT 1 FROM submissions WHERE file_hash = $1 AND user_id = $2 AND problem_id = $3 AND contest_id = $4)`
		args = []any{fileHash, userID, problemID, *contestID}
	}
	var exists bool
	err := q.QueryRow(ctx, query, args...).Scan(&exists)
	if err != nil {
		slog.ErrorContext(ctx, "submission: failed to check hash existence", "error", err)
		return false, apperror.NewInternal()
	}
	return exists, nil
}

// ── internals ─────────────────────────────────────────────────────────────────

const selectCols = `
	SELECT id, problem_id, user_id, contest_id, standing_id,
	       language, compiler, status, visibility, source_code_path,
	       file_hash, file_size, submitted_at,
	       judged_at, time_ms, memory_kb, compile_log,
	       COALESCE(problem_title, ''), COALESCE(problem_slug, '')
	FROM submissions`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRow(ctx context.Context, row rowScanner) (*domainSubmission.Submission, error) {
	var (
		id, userID, language, compiler, status, visibility, path string
		problemIDPtr                                              *string
		contestID, standingID                                     *string
		fileHash                                                  *string
		fileSize                                                  *int
		submittedAt                                               time.Time
		judgedAt                                                  *time.Time
		timeMs, memoryKb                                          *int
		compileLog                                                *string
		problemTitle, problemSlug                                 string
	)
	err := row.Scan(
		&id, &problemIDPtr, &userID, &contestID, &standingID,
		&language, &compiler, &status, &visibility, &path,
		&fileHash, &fileSize, &submittedAt,
		&judgedAt, &timeMs, &memoryKb, &compileLog,
		&problemTitle, &problemSlug,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		slog.ErrorContext(ctx, "submission: failed to scan row", "error", err)
		return nil, apperror.NewInternal()
	}

	problemID := ""
	if problemIDPtr != nil {
		problemID = *problemIDPtr
	}
	hash := ""
	if fileHash != nil {
		hash = *fileHash
	}
	size := 0
	if fileSize != nil {
		size = *fileSize
	}

	return domainSubmission.RestoreSubmission(
		id, problemID, shared.RestoreUserID(userID),
		contestID, standingID,
		domainSubmission.RestoreLanguage(language),
		compiler,
		domainSubmission.RestoreStatus(status),
		domainSubmission.RestoreVisibility(visibility),
		path, hash, size,
		submittedAt, judgedAt, timeMs, memoryKb, compileLog,
		problemTitle, problemSlug,
	), nil
}

func (r *Repository) UpdateVisibility(ctx context.Context, s *domainSubmission.Submission) error {
	q := infraPostgres.GetQuerier(ctx, r.db)
	tag, err := q.Exec(ctx, `
		UPDATE submissions SET visibility = $2 WHERE id = $1
	`, s.ID(), s.Visibility().String())
	if err != nil {
		slog.ErrorContext(ctx, "submission: failed to update visibility", "id", s.ID(), "error", err)
		return apperror.NewInternal()
	}
	if tag.RowsAffected() == 0 {
		return apperror.NewNotFound(domainSubmission.ErrCodeSubmissionNotFound, "submission not found")
	}
	return nil
}

func buildWhere(f domainSubmission.ListFilters) (string, []any) {
	where := " WHERE 1=1"
	args := []any{}
	i := 1
	if f.UserID != nil {
		where += fmt.Sprintf(" AND user_id = $%d", i)
		args = append(args, f.UserID.String())
		i++
	}
	if f.ProblemID != nil {
		where += fmt.Sprintf(" AND problem_id = $%d", i)
		args = append(args, *f.ProblemID)
		i++
	}
	if f.ContestID != nil {
		where += fmt.Sprintf(" AND contest_id = $%d", i)
		args = append(args, *f.ContestID)
		i++
	}
	if f.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", i)
		args = append(args, *f.Status)
		i++
	}
	if f.Language != nil {
		where += fmt.Sprintf(" AND language = $%d", i)
		args = append(args, *f.Language)
		i++
	}
	if f.Visibility != nil {
		where += fmt.Sprintf(" AND visibility = $%d", i)
		args = append(args, *f.Visibility)
		i++
	}
	if f.ExcludeContest {
		where += " AND contest_id IS NULL"
	}
	if f.SubmittedFrom != nil {
		where += fmt.Sprintf(" AND submitted_at >= $%d", i)
		args = append(args, *f.SubmittedFrom)
		i++
	}
	if f.SubmittedBefore != nil {
		where += fmt.Sprintf(" AND submitted_at < $%d", i)
		args = append(args, *f.SubmittedBefore)
		i++
	}
	return where, args
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nilIfZero(n int) *int {
	if n == 0 {
		return nil
	}
	return &n
}
