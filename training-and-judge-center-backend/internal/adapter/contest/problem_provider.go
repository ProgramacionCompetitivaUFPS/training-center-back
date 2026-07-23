package contest

import (
	"context"
	"log/slog"
	"strconv"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appContest "github.com/training-judge-center/backend/internal/application/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ProblemProvider struct {
	db infraPostgres.Querier
}

func NewProblemProvider(db infraPostgres.Querier) *ProblemProvider {
	return &ProblemProvider{db: db}
}

// FindBySlugs loads problems by slug and resolves CanAdd for the given caller.
// CanAdd = PUBLIC || isAdmin || (callerID = author_id) || callerID in problem_modifiers.
func (p *ProblemProvider) FindBySlugs(ctx context.Context, slugs []string, callerID string, isAdmin bool) (map[string]*appContest.ProblemInfo, error) {
	if len(slugs) == 0 {
		return map[string]*appContest.ProblemInfo{}, nil
	}
	q := infraPostgres.GetQuerier(ctx, p.db)

	// Build slug placeholders: $1, $2, ...
	args := make([]interface{}, 0, len(slugs)+2)
	for _, s := range slugs {
		args = append(args, s)
	}
	args = append(args, callerID, isAdmin)

	placeholder := ""
	for i := range slugs {
		if i > 0 {
			placeholder += ","
		}
		placeholder += "$" + strconv.Itoa(i+1)
	}
	callerPos := "$" + strconv.Itoa(len(slugs)+1)
	adminPos := "$" + strconv.Itoa(len(slugs)+2)

	query := `
		SELECT p.id, p.slug, p.title, p.status,
		       p.accessibility,
		       (` + adminPos + `::boolean
		        OR p.accessibility = 'PUBLIC'
		        OR p.author_id = ` + callerPos + `::uuid
		        OR ` + callerPos + `::uuid = ANY(p.modifiers_ids)
		        ) AS can_add
		FROM problems p
		WHERE p.slug IN (` + placeholder + `)`

	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		slog.ErrorContext(ctx, "FindBySlugs query failed", "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	result := make(map[string]*appContest.ProblemInfo, len(slugs))
	for rows.Next() {
		var id, slug, title, status, accessibility string
		var canAdd bool
		if err := rows.Scan(&id, &slug, &title, &status, &accessibility, &canAdd); err != nil {
			slog.ErrorContext(ctx, "failed to scan problem row", "error", err)
			return nil, apperror.NewInternal()
		}
		result[slug] = &appContest.ProblemInfo{
			ID:          id,
			Slug:        slug,
			Title:       title,
			IsPublished: status == "PUBLISHED",
			CanAdd:      canAdd,
		}
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "FindBySlugs rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}

// FindByIDsWithLimits loads problem info including time_limit and memory_limit.
func (p *ProblemProvider) FindByIDsWithLimits(ctx context.Context, ids []string) (map[string]*appContest.ProblemWithLimits, error) {
	if len(ids) == 0 {
		return map[string]*appContest.ProblemWithLimits{}, nil
	}
	q := infraPostgres.GetQuerier(ctx, p.db)

	args := make([]interface{}, len(ids))
	placeholder := ""
	for i, id := range ids {
		args[i] = id
		if i > 0 {
			placeholder += ","
		}
		placeholder += "$" + strconv.Itoa(i+1)
	}

	rows, err := q.Query(ctx, `SELECT id, slug, title, time_limit, memory_limit FROM problems WHERE id IN (`+placeholder+`)`, args...)
	if err != nil {
		slog.ErrorContext(ctx, "FindByIDsWithLimits query failed", "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	result := make(map[string]*appContest.ProblemWithLimits, len(ids))
	for rows.Next() {
		var id, slug, title string
		var timeLimit, memoryLimit int
		if err := rows.Scan(&id, &slug, &title, &timeLimit, &memoryLimit); err != nil {
			slog.ErrorContext(ctx, "failed to scan problem with limits", "error", err)
			return nil, apperror.NewInternal()
		}
		result[id] = &appContest.ProblemWithLimits{
			ID:          id,
			Slug:        slug,
			Title:       title,
			TimeLimit:   timeLimit,
			MemoryLimit: memoryLimit,
		}
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "FindByIDsWithLimits rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}

// FindByIDs loads basic problem info (id, slug, title) by problem UUID.
func (p *ProblemProvider) FindByIDs(ctx context.Context, ids []string) (map[string]*appContest.ProblemBasicInfo, error) {
	if len(ids) == 0 {
		return map[string]*appContest.ProblemBasicInfo{}, nil
	}
	q := infraPostgres.GetQuerier(ctx, p.db)

	args := make([]interface{}, len(ids))
	placeholder := ""
	for i, id := range ids {
		args[i] = id
		if i > 0 {
			placeholder += ","
		}
		placeholder += "$" + strconv.Itoa(i+1)
	}

	rows, err := q.Query(ctx, `SELECT id, slug, title FROM problems WHERE id IN (`+placeholder+`)`, args...)
	if err != nil {
		slog.ErrorContext(ctx, "FindByIDs query failed", "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	result := make(map[string]*appContest.ProblemBasicInfo, len(ids))
	for rows.Next() {
		var id, slug, title string
		if err := rows.Scan(&id, &slug, &title); err != nil {
			slog.ErrorContext(ctx, "failed to scan problem basic info", "error", err)
			return nil, apperror.NewInternal()
		}
		result[id] = &appContest.ProblemBasicInfo{ID: id, Slug: slug, Title: title}
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "FindByIDs rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}
