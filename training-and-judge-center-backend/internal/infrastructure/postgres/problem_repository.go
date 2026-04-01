package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ProblemRepository struct {
	db *pgxpool.Pool
}

func NewProblemRepository(db *pgxpool.Pool) *ProblemRepository {
	return &ProblemRepository{db: db}
}

func (r *ProblemRepository) Save(ctx context.Context, p *problem.Problem) error {
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

	slug := p.Slug.String()
	title := p.Title.String()
	status := p.Status.String()
	accessibility := p.Accessibility.String()

	langOverridesJSON, err := mapLangOverridesToDB(p.LangOverrides)
	if err != nil {
		slog.ErrorContext(ctx, "error mapping lang overrides to DB", "error", err, "problem_id", p.ID)
		return apperror.NewInternal()
	}

	solutionsJSON, err := mapSolutionsToDB(p.Solutions)
	if err != nil {
		slog.ErrorContext(ctx, "error mapping solutions to DB", "error", err, "problem_id", p.ID)
		return apperror.NewInternal()
	}

	checkerJSON, err := mapJudgingFileToDB(p.Checker)
	if err != nil {
		slog.ErrorContext(ctx, "error mapping checker to DB", "error", err, "problem_id", p.ID)
		return apperror.NewInternal()
	}

	validatorJSON, err := mapJudgingFileToDB(p.Validator)
	if err != nil {
		slog.ErrorContext(ctx, "error mapping validator to DB", "error", err, "problem_id", p.ID)
		return apperror.NewInternal()
	}

	var tl *int
	if p.TimeLimit != nil {
		v := p.TimeLimit.Value()
		tl = &v
	}

	var ml *int
	if p.MemoryLimit != nil {
		v := p.MemoryLimit.Value()
		ml = &v
	}

	_, err = r.db.Exec(ctx, query,
		p.ID,
		slug,
		title,
		p.Statement.Value(),
		tl,
		ml,
		p.Tags.Values(),
		status,
		accessibility,
		p.AuthorID,
		p.ModifierIDs,
		langOverridesJSON,
		p.TestCasesKey,
		solutionsJSON,
		checkerJSON,
		validatorJSON,
		p.JudgingUpdatedAt,
		p.CreatedAt,
		p.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "problems_slug_key" {
			return &problem.ErrSlugAlreadyExists{Slug: slug}
		}
		slog.ErrorContext(ctx, "Database error in Save", "error", err, "problem_id", p.ID, "slug", slug)
		return apperror.NewInternal()
	}

	return nil
}

func (r *ProblemRepository) FindBySlug(ctx context.Context, slug problem.Slug) (*problem.Problem, error) {
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

func scanProblem(row pgx.Row) (*problem.Problem, error) {
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

	return problem.RestoreProblem(
		pId,
		pSlug,
		pTitle,
		pStatement,
		pTl,
		pMl,
		pTags,
		pStatus,
		pAccessibility,
		pAuthorId,
		pModifiers,
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

func (r *ProblemRepository) ExistsBySlug(ctx context.Context, slug problem.Slug) (bool, error) {
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

func mapLangOverridesToDB(overrides []problem.LanguageOverride) ([]byte, error) {
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

func mapSolutionsToDB(solutions []problem.JudgingFile) ([]byte, error) {
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

func mapJudgingFileToDB(file *problem.JudgingFile) ([]byte, error) {
	if file == nil {
		return nil, nil // SQL NULL
	}
	return json.Marshal(dbJudgingFile{
		Filename: file.Filename(),
		FileKey:  file.FileKey(),
		Language: file.Language(),
	})
}

func langOverridesFromDB(data []byte) ([]problem.LanguageOverride, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var dbOverrides []dbLangOverride
	if err := json.Unmarshal(data, &dbOverrides); err != nil {
		return nil, err
	}
	overrides := make([]problem.LanguageOverride, len(dbOverrides))
	for i, lo := range dbOverrides {
		overrides[i] = problem.RestoreLanguageOverride(lo.Language, lo.TimeLimit, lo.MemoryLimit)
	}
	return overrides, nil
}

func solutionsFromDB(data []byte) ([]problem.JudgingFile, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var dbSolutions []dbJudgingFile
	if err := json.Unmarshal(data, &dbSolutions); err != nil {
		return nil, err
	}
	solutions := make([]problem.JudgingFile, len(dbSolutions))
	for i, sol := range dbSolutions {
		solutions[i] = problem.RestoreJudgingFile(sol.Filename, sol.FileKey, sol.Language)
	}
	return solutions, nil
}

func judgingFileFromDB(data []byte) (*problem.JudgingFile, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var dbFile dbJudgingFile
	if err := json.Unmarshal(data, &dbFile); err != nil {
		return nil, err
	}
	j := problem.RestoreJudgingFile(dbFile.Filename, dbFile.FileKey, dbFile.Language)
	return &j, nil
}
