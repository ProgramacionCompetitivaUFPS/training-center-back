package postgres

import (
	"context"
	"encoding/json"
	"errors"

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
			tags, status, accessibility, author_id, modifiers_ids, lang_overrides, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
	`

	slug := p.Slug.String()
	title := p.Title.String()
	status := p.Status.String()
	accessibility := p.Accessibility.String()

	type langOverrideDB struct {
		Language    string `json:"language"`
		TimeLimit   *int   `json:"timeLimit,omitempty"`
		MemoryLimit *int   `json:"memoryLimit,omitempty"`
	}
	dbOverrides := make([]langOverrideDB, 0, len(p.LangOverrides))
	for _, lo := range p.LangOverrides {
		dbOverrides = append(dbOverrides, langOverrideDB{
			Language:    lo.Language(),
			TimeLimit:   lo.TimeLimit(),
			MemoryLimit: lo.MemoryLimit(),
		})
	}

	langOverridesJSON, err := json.Marshal(dbOverrides)
	if err != nil {
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
		p.Statement,
		tl,
		ml,
		p.Tags.Values(),
		status,
		accessibility,
		p.AuthorID,
		p.ModifierIDs,
		langOverridesJSON,
		p.CreatedAt,
		p.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "problems_slug_key" {
			return &problem.ErrSlugAlreadyExists{Slug: slug}
		}
		return apperror.NewInternal()
	}

	return nil
}

func (r *ProblemRepository) FindBySlug(ctx context.Context, slug problem.Slug) (*problem.Problem, error) {
	return nil, apperror.NewInternal()
}

func (r *ProblemRepository) ExistsBySlug(ctx context.Context, slug problem.Slug) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM problems WHERE slug = $1)`
	
	var exists bool
	err := r.db.QueryRow(ctx, query, slug.String()).Scan(&exists)
	if err != nil {
		return false, apperror.NewInternal()
	}

	return exists, nil
}
