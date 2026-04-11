package problem

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ProblemRepository struct {
	db *pgxpool.Pool
}

func NewProblemRepository(db *pgxpool.Pool) *ProblemRepository {
	return &ProblemRepository{db: db}
}

func (r *ProblemRepository) Save(ctx context.Context, p *domainProblem.Problem) error {
	query := `
		INSERT INTO problems (
			id, slug, title, statement, time_limit, memory_limit,
			tags, status, accessibility, author_id, modifiers_ids, lang_overrides,
			test_cases_key, solutions, checker, validator, judging_updated_at,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
		) ON CONFLICT (id) DO UPDATE SET
			slug = EXCLUDED.slug,
			title = EXCLUDED.title,
			statement = EXCLUDED.statement,
			time_limit = EXCLUDED.time_limit,
			memory_limit = EXCLUDED.memory_limit,
			tags = EXCLUDED.tags,
			status = EXCLUDED.status,
			accessibility = EXCLUDED.accessibility,
			modifiers_ids = EXCLUDED.modifiers_ids,
			lang_overrides = EXCLUDED.lang_overrides,
			test_cases_key = EXCLUDED.test_cases_key,
			solutions = EXCLUDED.solutions,
			checker = EXCLUDED.checker,
			validator = EXCLUDED.validator,
			judging_updated_at = EXCLUDED.judging_updated_at,
			updated_at = EXCLUDED.updated_at
	`

	slug := p.Slug().String()
	title := p.Title().String()
	status := p.Status().String()
	accessibility := p.Accessibility().String()

	langOverridesJSON, err := mapLangOverridesToDB(p.LangOverrides())
	if err != nil {
		slog.ErrorContext(ctx, "error mapping lang overrides to DB", "error", err, "problem_id", p.ID())
		return apperror.NewInternal()
	}

	solutionsJSON, err := mapSolutionsToDB(p.Solutions())
	if err != nil {
		slog.ErrorContext(ctx, "error mapping solutions to DB", "error", err, "problem_id", p.ID())
		return apperror.NewInternal()
	}

	checkerJSON, err := mapJudgingFileToDB(p.Checker())
	if err != nil {
		slog.ErrorContext(ctx, "error mapping checker to DB", "error", err, "problem_id", p.ID())
		return apperror.NewInternal()
	}

	validatorJSON, err := mapJudgingFileToDB(p.Validator())
	if err != nil {
		slog.ErrorContext(ctx, "error mapping validator to DB", "error", err, "problem_id", p.ID())
		return apperror.NewInternal()
	}

	var tl *int
	if p.TimeLimit() != nil {
		v := p.TimeLimit().Milliseconds()
		tl = &v
	}

	var ml *int
	if p.MemoryLimit() != nil {
		v := p.MemoryLimit().Megabytes()
		ml = &v
	}

	modifierStrings := make([]string, len(p.ModifierIDs()))
	for i, id := range p.ModifierIDs() {
		modifierStrings[i] = id.Value()
	}

	_, err = r.db.Exec(ctx, query,
		p.ID(),
		slug,
		title,
		p.Statement().Value(),
		tl,
		ml,
		p.Tags().Values(),
		status,
		accessibility,
		p.AuthorID().Value(),
		modifierStrings,
		langOverridesJSON,
		p.TestCasesKey(),
		solutionsJSON,
		checkerJSON,
		validatorJSON,
		p.JudgingUpdatedAt(),
		p.CreatedAt(),
		p.UpdatedAt(),
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "problems_slug_key" {
			return &domainProblem.ErrSlugAlreadyExists{Slug: slug}
		}
		slog.ErrorContext(ctx, "Database error in Save", "error", err, "problem_id", p.ID(), "slug", slug)
		return apperror.NewInternal()
	}

	return nil
}

func (r *ProblemRepository) FindBySlug(ctx context.Context, slug domainProblem.Slug) (*domainProblem.Problem, error) {
	query := `
		SELECT id, slug, title, statement, time_limit, memory_limit,
			tags, status, accessibility, author_id, modifiers_ids, lang_overrides,
			test_cases_key, solutions, checker, validator, judging_updated_at,
			created_at, updated_at
		FROM problems
		WHERE slug = $1
	`
	row := r.db.QueryRow(ctx, query, slug.String())
	return scanProblem(row)
}

func scanProblem(row pgx.Row) (*domainProblem.Problem, error) {
	var pId, pSlug, pTitle, pStatus, pAccessibility, pAuthorId string
	var pStatement, pTestCasesKey *string
	var pTl, pMl *int
	var pTags, pModifiers []string
	var pLangOverridesJSON, pSolutionsJSON, pCheckerJSON, pValidatorJSON []byte
	var pJudgingUpdatedAt *time.Time
	var pCreatedAt, pUpdatedAt time.Time

	err := row.Scan(
		&pId,
		&pSlug,
		&pTitle,
		&pStatement,
		&pTl,
		&pMl,
		&pTags,
		&pStatus,
		&pAccessibility,
		&pAuthorId,
		&pModifiers,
		&pLangOverridesJSON,
		&pTestCasesKey,
		&pSolutionsJSON,
		&pCheckerJSON,
		&pValidatorJSON,
		&pJudgingUpdatedAt,
		&pCreatedAt,
		&pUpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "problem not found")
		}
		slog.Error("Database error in scanProblem", "error", err)
		return nil, apperror.NewInternal()
	}

	langOverrides, err := langOverridesFromDB(pLangOverridesJSON)
	if err != nil {
		slog.Error("error mapping lang overrides from DB", "error", err, "problem_id", pId)
		return nil, apperror.NewInternal()
	}
	solutions, err := solutionsFromDB(pSolutionsJSON)
	if err != nil {
		slog.Error("error mapping field from DB", "error", err, "problem_id", pId)
		return nil, apperror.NewInternal()
	}
	checker, err := judgingFileFromDB(pCheckerJSON)
	if err != nil {
		slog.Error("error mapping field from DB", "error", err, "problem_id", pId)
		return nil, apperror.NewInternal()
	}
	validator, err := judgingFileFromDB(pValidatorJSON)
	if err != nil {
		slog.Error("error mapping validator from DB", "error", err, "problem_id", pId)
		return nil, apperror.NewInternal()
	}

	modifierIDs := make([]shared.UserID, len(pModifiers))
	for i, id := range pModifiers {
		modifierIDs[i] = shared.RestoreUserID(id)
	}

	return domainProblem.RestoreProblem(
		pId,
		pSlug,
		pTitle,
		pStatement,
		pTl,
		pMl,
		pTags,
		pStatus,
		pAccessibility,
		shared.RestoreUserID(pAuthorId),
		modifierIDs,
		langOverrides,
		pTestCasesKey,
		solutions,
		checker,
		validator,
		pJudgingUpdatedAt,
		pCreatedAt,
		pUpdatedAt,
	), nil
}

func (r *ProblemRepository) ExistsBySlug(ctx context.Context, slug domainProblem.Slug) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM problems WHERE slug = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, slug.String()).Scan(&exists)
	if err != nil {
		slog.ErrorContext(ctx, "Database error in ExistsBySlug", "error", err, "slug", slug.String())
		return false, apperror.NewInternal()
	}

	return exists, nil
}

type dbJudgingFile struct {
	Filename string `json:"filename"`
	FileKey  string `json:"fileKey"`
	Language string `json:"language"`
}

type dbLangOverride struct {
	Language    string `json:"language"`
	TimeLimit   *int   `json:"timeLimit,omitempty"`
	MemoryLimit *int   `json:"memoryLimit,omitempty"`
}

func mapLangOverridesToDB(overrides []domainProblem.LanguageOverride) ([]byte, error) {
	dbOverrides := make([]dbLangOverride, 0, len(overrides))
	for _, lo := range overrides {
		dbOverrides = append(dbOverrides, dbLangOverride{
			Language:    lo.Language(),
			TimeLimit:   lo.TimeLimit(),
			MemoryLimit: lo.MemoryLimit(),
		})
	}
	return json.Marshal(dbOverrides)
}

func mapSolutionsToDB(solutions []domainProblem.JudgingFile) ([]byte, error) {
	dbSolutions := make([]dbJudgingFile, 0, len(solutions))
	for _, sol := range solutions {
		dbSolutions = append(dbSolutions, dbJudgingFile{
			Filename: sol.Filename(),
			FileKey:  sol.FileKey(),
			Language: sol.Language(),
		})
	}
	return json.Marshal(dbSolutions)
}

func mapJudgingFileToDB(file *domainProblem.JudgingFile) ([]byte, error) {
	if file == nil {
		return nil, nil // SQL NULL
	}
	return json.Marshal(dbJudgingFile{
		Filename: file.Filename(),
		FileKey:  file.FileKey(),
		Language: file.Language(),
	})
}

func langOverridesFromDB(data []byte) ([]domainProblem.LanguageOverride, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var dbOverrides []dbLangOverride
	if err := json.Unmarshal(data, &dbOverrides); err != nil {
		return nil, err
	}
	overrides := make([]domainProblem.LanguageOverride, len(dbOverrides))
	for i, lo := range dbOverrides {
		overrides[i] = domainProblem.RestoreLanguageOverride(lo.Language, lo.TimeLimit, lo.MemoryLimit)
	}
	return overrides, nil
}

func solutionsFromDB(data []byte) ([]domainProblem.JudgingFile, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var dbSolutions []dbJudgingFile
	if err := json.Unmarshal(data, &dbSolutions); err != nil {
		return nil, err
	}
	solutions := make([]domainProblem.JudgingFile, len(dbSolutions))
	for i, sol := range dbSolutions {
		solutions[i] = domainProblem.RestoreJudgingFile(sol.Filename, sol.FileKey, sol.Language)
	}
	return solutions, nil
}

func judgingFileFromDB(data []byte) (*domainProblem.JudgingFile, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var dbFile dbJudgingFile
	if err := json.Unmarshal(data, &dbFile); err != nil {
		return nil, err
	}
	j := domainProblem.RestoreJudgingFile(dbFile.Filename, dbFile.FileKey, dbFile.Language)
	return &j, nil
}

func (r *ProblemRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM problems WHERE id = $1`, id)
	if err != nil {
		slog.ErrorContext(ctx, "Database error in Delete", "error", err, "problem_id", id)
		return apperror.NewInternal()
	}
	if tag.RowsAffected() == 0 {
		return apperror.NewNotFound(apperror.ErrCodeNotFound, "problem not found")
	}
	return nil
}

func (r *ProblemRepository) List(ctx context.Context, filters domainProblem.ListFilters) ([]*domainProblem.Problem, int, error) {
	var conds []string
	var args []any
	idx := 1

	nextArg := func(v any) string {
		args = append(args, v)
		s := fmt.Sprintf("$%d", idx)
		idx++
		return s
	}

	if len(filters.Statuses) > 0 {
		statusStrings := make([]string, len(filters.Statuses))
		for i, s := range filters.Statuses {
			statusStrings[i] = s.String()
		}
		conds = append(conds, fmt.Sprintf("status = ANY(%s)", nextArg(statusStrings)))
	}

	if filters.ViewerModifierID != nil {
		p := nextArg(*filters.ViewerModifierID)
		conds = append(conds, fmt.Sprintf(
			"((status = 'PUBLISHED' AND accessibility = 'PUBLIC') OR author_id = %s OR %s = ANY(modifiers_ids))",
			p, p,
		))
	}

	if filters.AuthorID != nil {
		conds = append(conds, fmt.Sprintf("author_id = %s", nextArg(*filters.AuthorID)))
	}

	if len(filters.Tags) > 0 {
		conds = append(conds, fmt.Sprintf("tags @> %s", nextArg(filters.Tags)))
	}

	if filters.Accessibility != nil {
		conds = append(conds, fmt.Sprintf("accessibility = %s", nextArg(filters.Accessibility.String())))
	}

	var where string
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	countArgs := make([]any, len(args))
	copy(countArgs, args)

	offset := (filters.Page - 1) * filters.Limit
	limitArg := nextArg(filters.Limit)
	offsetArg := nextArg(offset)

	selectQuery := fmt.Sprintf(`
		SELECT id, slug, title, statement, time_limit, memory_limit,
			tags, status, accessibility, author_id, modifiers_ids, lang_overrides,
			test_cases_key, solutions, checker, validator, judging_updated_at,
			created_at, updated_at
		FROM problems
		%s
		ORDER BY created_at DESC
		LIMIT %s OFFSET %s
	`, where, limitArg, offsetArg)

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM problems %s", where)

	g, gCtx := errgroup.WithContext(ctx)

	var result []*domainProblem.Problem
	var total int

	g.Go(func() error {
		rows, queryErr := r.db.Query(gCtx, selectQuery, args...)
		if queryErr != nil {
			return queryErr
		}
		defer rows.Close()
		for rows.Next() {
			p, err := scanProblem(rows)
			if err != nil {
				return err
			}
			result = append(result, p)
		}
		return rows.Err()
	})

	g.Go(func() error {
		var countErr error
		countErr = r.db.QueryRow(gCtx, countQuery, countArgs...).Scan(&total)
		return countErr
	})

	if err := g.Wait(); err != nil {
		slog.ErrorContext(ctx, "Database error in List", "error", err)
		return nil, 0, apperror.NewInternal()
	}

	if result == nil {
		result = []*domainProblem.Problem{}
	}

	return result, total, nil
}
