package material

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Save(ctx context.Context, m *material.Material) error {
	query := `
		INSERT INTO materials (
			id, group_id, author_id, title, content, tags,
			status, pinned, pinned_at, created_at, updated_at, published_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		) ON CONFLICT (id) DO UPDATE SET
			title        = EXCLUDED.title,
			content      = EXCLUDED.content,
			tags         = EXCLUDED.tags,
			status       = EXCLUDED.status,
			pinned       = EXCLUDED.pinned,
			pinned_at    = EXCLUDED.pinned_at,
			updated_at   = EXCLUDED.updated_at,
			published_at = EXCLUDED.published_at
	`

	_, err := r.db.Exec(ctx, query,
		m.ID(),
		m.GroupID(),
		m.AuthorID().Value(),
		m.Title().String(),
		m.Content().String(),
		m.Tags().Values(),
		m.Status().String(),
		m.Pinned(),
		m.PinnedAt(),
		m.CreatedAt(),
		m.UpdatedAt(),
		m.PublishedAt(),
	)
	if err != nil {
		slog.ErrorContext(ctx, "database error in Save", "error", err, "material_id", m.ID())
		return apperror.NewInternal()
	}
	return nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (*material.Material, error) {
	query := `
		SELECT id, group_id, author_id, title, content, tags,
		       status, pinned, pinned_at, created_at, updated_at, published_at
		FROM materials
		WHERE id = $1
	`
	row := r.db.QueryRow(ctx, query, id)
	m, err := scanMaterial(ctx, row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound(material.ErrCodeMaterialNotFound, "material not found")
		}
		slog.ErrorContext(ctx, "database error in FindByID", "error", err, "material_id", id)
		return nil, apperror.NewInternal()
	}
	return m, nil
}

func (r *Repository) List(ctx context.Context, groupID string, filters material.ListFilters) ([]*material.Material, int, error) {
	var conds []string
	var args []any
	idx := 1

	nextArg := func(v any) string {
		args = append(args, v)
		s := fmt.Sprintf("$%d", idx)
		idx++
		return s
	}

	conds = append(conds, fmt.Sprintf("group_id = %s", nextArg(groupID)))

	if len(filters.Statuses) > 0 {
		statusStrings := make([]string, len(filters.Statuses))
		for i, s := range filters.Statuses {
			statusStrings[i] = s.String()
		}
		conds = append(conds, fmt.Sprintf("status = ANY(%s)", nextArg(statusStrings)))
	}

	if filters.ViewerID != nil {
		p := nextArg(*filters.ViewerID)
		conds = append(conds, fmt.Sprintf("(status = 'PUBLISHED' OR author_id = %s)", p))
	}

	if filters.AuthorID != nil {
		conds = append(conds, fmt.Sprintf("author_id = %s", nextArg(*filters.AuthorID)))
	}

	if len(filters.Tags) > 0 {
		conds = append(conds, fmt.Sprintf("tags @> %s", nextArg(filters.Tags)))
	}

	if filters.Pinned != nil {
		conds = append(conds, fmt.Sprintf("pinned = %s", nextArg(*filters.Pinned)))
	}

	if filters.PublishedFrom != nil {
		conds = append(conds, fmt.Sprintf("published_at >= %s", nextArg(*filters.PublishedFrom)))
	}

	if filters.PublishedTo != nil {
		// Add 1 day so the upper bound covers the entire boundary date (inclusive spec).
		to := filters.PublishedTo.AddDate(0, 0, 1)
		conds = append(conds, fmt.Sprintf("published_at < %s", nextArg(to)))
	}

	// FTS: store the $N ref so it can be reused in ORDER BY without a second parameter.
	var ftsRef string
	const ftsVector = `(setweight(to_tsvector('simple', title), 'A') || setweight(to_tsvector('simple', COALESCE(content, '')), 'B'))`
	if filters.SearchQuery != nil && *filters.SearchQuery != "" {
		ftsRef = nextArg(*filters.SearchQuery)
		conds = append(conds, fmt.Sprintf("%s @@ plainto_tsquery('simple', %s)", ftsVector, ftsRef))
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	countArgs := make([]any, len(args))
	copy(countArgs, args)

	offset := (filters.Page - 1) * filters.Limit
	limitArg := nextArg(filters.Limit)
	offsetArg := nextArg(offset)

	orderBy := listOrderBy(filters.SortBy, ftsRef, ftsVector)

	selectQuery := fmt.Sprintf(`
		SELECT id, group_id, author_id, title, content, tags,
		       status, pinned, pinned_at, created_at, updated_at, published_at
		FROM materials
		%s
		ORDER BY %s
		LIMIT %s OFFSET %s
	`, where, orderBy, limitArg, offsetArg)

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM materials %s", where)

	g, gCtx := errgroup.WithContext(ctx)

	var result []*material.Material
	var total int

	g.Go(func() error {
		rows, queryErr := r.db.Query(gCtx, selectQuery, args...)
		if queryErr != nil {
			return queryErr
		}
		defer rows.Close()
		for rows.Next() {
			m, err := scanMaterial(gCtx, rows)
			if err != nil {
				return fmt.Errorf("scanning material row: %w", err)
			}
			result = append(result, m)
		}
		return rows.Err()
	})

	g.Go(func() error {
		return r.db.QueryRow(gCtx, countQuery, countArgs...).Scan(&total)
	})

	if err := g.Wait(); err != nil {
		slog.ErrorContext(ctx, "database error in List", "error", err, "group_id", groupID)
		return nil, 0, apperror.NewInternal()
	}

	if result == nil {
		result = []*material.Material{}
	}

	return result, total, nil
}

// listOrderBy builds the ORDER BY clause.
// When a search query is active, pinned boost is suppressed (FR-032).
func listOrderBy(sortBy material.SortField, ftsRef, ftsVector string) string {
	switch sortBy {
	case material.SortByRelevance:
		if ftsRef != "" {
			return fmt.Sprintf(
				"ts_rank(%s, plainto_tsquery('simple', %s)) DESC, published_at DESC NULLS LAST, created_at DESC",
				ftsVector, ftsRef,
			)
		}
		// relevance without query → traditional order
		return "pinned DESC, pinned_at DESC NULLS LAST, published_at DESC NULLS LAST, created_at DESC"
	case material.SortByTitle:
		return "title ASC, published_at DESC NULLS LAST, created_at DESC"
	default: // SortByPublishedAt
		if ftsRef != "" {
			// Searching: no pinned boost (FR-032)
			return "published_at DESC NULLS LAST, created_at DESC"
		}
		return "pinned DESC, pinned_at DESC NULLS LAST, published_at DESC NULLS LAST, created_at DESC"
	}
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM materials WHERE id = $1`, id)
	if err != nil {
		slog.ErrorContext(ctx, "database error in Delete", "error", err, "material_id", id)
		return apperror.NewInternal()
	}
	if tag.RowsAffected() == 0 {
		return apperror.NewNotFound(material.ErrCodeMaterialNotFound, "material not found")
	}
	return nil
}

func scanMaterial(ctx context.Context, row pgx.Row) (*material.Material, error) {
	var id, groupID, authorID, title, content, status string
	var tags []string
	var pinned bool
	var pinnedAt, publishedAt *time.Time
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&id, &groupID, &authorID, &title, &content, &tags,
		&status, &pinned, &pinnedAt, &createdAt, &updatedAt, &publishedAt,
	)
	if err != nil {
		return nil, err
	}

	m := material.RestoreMaterial(
		id,
		groupID,
		shared.RestoreUserID(authorID),
		title,
		content,
		tags,
		status,
		pinned,
		pinnedAt,
		createdAt,
		updatedAt,
		publishedAt,
	)
	if !m.Status().IsDraft() && !m.Status().IsPublished() {
		slog.WarnContext(ctx, "unrecognised material status restored from DB", "material_id", id, "status", status)
	}
	return m, nil
}
