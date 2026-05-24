package group

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"golang.org/x/sync/errgroup"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Save(ctx context.Context, g *domainGroup.Group) error {
	query := `
		INSERT INTO groups (
			id, name, description, visibility, join_policy,
			is_default, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`
	_, err := infraPostgres.GetQuerier(ctx, r.db).Exec(ctx, query,
		g.ID(),
		g.Name().Value(),
		g.Description(),
		g.Visibility().String(),
		g.JoinPolicy().String(),
		g.IsDefault(),
		g.CreatedBy().Value(),
		g.CreatedAt(),
		g.UpdatedAt(),
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == infraPostgres.UniqueViolation {
			return apperror.NewConflict(domainGroup.ErrCodeNameAlreadyExists, "A group with this name already exists")
		}
		slog.ErrorContext(ctx, "failed to save group", "error", err, "group_id", g.ID())
		return apperror.NewInternal()
	}
	return nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (*domainGroup.Group, error) {
	const q = `
		SELECT id, name, description, visibility, join_policy, is_default,
		       created_by, created_at, updated_at
		FROM groups WHERE id = $1
	`
	row := r.db.QueryRow(ctx, q, id)
	g, err := scanGroup(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound(domainGroup.ErrCodeGroupNotFound, "group not found")
		}
		slog.ErrorContext(ctx, "FindByID failed", "error", err, "group_id", id)
		return nil, apperror.NewInternal()
	}
	return g, nil
}

func (r *Repository) ExistsByName(ctx context.Context, name domainGroup.GroupName) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM groups WHERE LOWER(name) = LOWER($1))`,
		name.Value(),
	).Scan(&exists)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check group name existence", "error", err)
		return false, apperror.NewInternal()
	}
	return exists, nil
}

func (r *Repository) FindDefault(ctx context.Context) (*domainGroup.Group, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, name, description, visibility, join_policy,
		       is_default, created_by, created_at, updated_at
		FROM groups
		WHERE is_default = TRUE
		LIMIT 1
	`)
	g, err := scanGroup(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound(domainGroup.ErrCodeGroupNotFound, "group not found")
		}
		slog.ErrorContext(ctx, "FindDefault failed", "error", err)
		return nil, apperror.NewInternal()
	}
	return g, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM groups WHERE id = $1`, id)
	if err != nil {
		slog.ErrorContext(ctx, "failed to delete group", "error", err, "group_id", id)
		return apperror.NewInternal()
	}
	if tag.RowsAffected() == 0 {
		return apperror.NewNotFound(domainGroup.ErrCodeGroupNotFound, "group not found")
	}
	return nil
}

func (r *Repository) List(ctx context.Context, filters domainGroup.ListFilters) ([]*domainGroup.Group, int, error) {
	var conds []string
	var args []any
	idx := 1
	nextArg := func(v any) string {
		args = append(args, v)
		s := fmt.Sprintf("$%d", idx)
		idx++
		return s
	}

	// Register viewerID upfront when it may appear in WHERE or ORDER BY,
	// so its placeholder index is stable regardless of condition order.
	viewerArg := func() string { return "" }
	if filters.OnlyMyGroups != nil || !filters.ViewerIsAdmin || filters.SortBy == domainGroup.SortByJoinedAt {
		placeholder := nextArg(filters.ViewerID.Value())
		viewerArg = func() string { return placeholder }
	}

	if filters.OnlyMyGroups != nil {
		if filters.OnlyMyGroups.RoleFilter != nil {
			roleArg := nextArg(filters.OnlyMyGroups.RoleFilter.String())
			conds = append(conds, fmt.Sprintf(
				"EXISTS (SELECT 1 FROM group_members gm WHERE gm.group_id = g.id AND gm.user_id = %s AND gm.member_role = %s)",
				viewerArg(), roleArg,
			))
		} else {
			conds = append(conds, fmt.Sprintf(
				"EXISTS (SELECT 1 FROM group_members gm WHERE gm.group_id = g.id AND gm.user_id = %s)",
				viewerArg(),
			))
		}
	} else if !filters.ViewerIsAdmin {
		conds = append(conds, fmt.Sprintf(
			"(g.visibility = 'VISIBLE' OR EXISTS (SELECT 1 FROM group_members gm WHERE gm.group_id = g.id AND gm.user_id = %s))",
			viewerArg(),
		))
	}

	if filters.ExcludeDefault {
		conds = append(conds, "g.is_default = FALSE")
	}

	if filters.Search != "" {
		arg := nextArg("%" + strings.ToLower(filters.Search) + "%")
		conds = append(conds, fmt.Sprintf("(LOWER(g.name) LIKE %s OR LOWER(COALESCE(g.description, '')) LIKE %s)", arg, arg))
	}

	if filters.Visibility != nil {
		conds = append(conds, fmt.Sprintf("g.visibility = %s", nextArg(filters.Visibility.String())))
	}

	if filters.JoinPolicy != nil {
		conds = append(conds, fmt.Sprintf("g.join_policy = %s", nextArg(filters.JoinPolicy.String())))
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	orderBy := mapSortClause(filters, viewerArg)

	memberCountSelect := ""
	if filters.SortBy == domainGroup.SortByMemberCount {
		memberCountSelect = ", (SELECT COUNT(*) FROM group_members gm2 WHERE gm2.group_id = g.id) AS mc"
	}

	countArgs := make([]any, len(args))
	copy(countArgs, args)

	offset := (filters.Page - 1) * filters.Limit
	limitArg := nextArg(filters.Limit)
	offsetArg := nextArg(offset)

	selectQuery := fmt.Sprintf(`
		SELECT g.id, g.name, g.description, g.visibility, g.join_policy, g.is_default,
		       g.created_by, g.created_at, g.updated_at%s
		FROM groups g
		%s
		ORDER BY %s
		LIMIT %s OFFSET %s
	`, memberCountSelect, where, orderBy, limitArg, offsetArg)

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM groups g %s", where)

	eg, gCtx := errgroup.WithContext(ctx)
	var result []*domainGroup.Group
	var total int

	eg.Go(func() error {
		rows, err := r.db.Query(gCtx, selectQuery, args...)
		if err != nil {
			slog.ErrorContext(gCtx, "database error querying groups", "error", err)
			return apperror.NewInternal()
		}
		defer rows.Close()
		for rows.Next() {
			g, err := scanGroupRow(rows, memberCountSelect != "")
			if err != nil {
				slog.ErrorContext(gCtx, "database error scanning group row", "error", err)
				return apperror.NewInternal()
			}
			result = append(result, g)
		}
		if err := rows.Err(); err != nil {
			slog.ErrorContext(gCtx, "database error iterating group rows", "error", err)
			return apperror.NewInternal()
		}
		return nil
	})

	eg.Go(func() error {
		if err := r.db.QueryRow(gCtx, countQuery, countArgs...).Scan(&total); err != nil {
			slog.ErrorContext(gCtx, "database error counting groups", "error", err)
			return apperror.NewInternal()
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, 0, err // already apperror — goroutines already logged
	}
	if result == nil {
		result = []*domainGroup.Group{}
	}
	return result, total, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanGroup(row rowScanner) (*domainGroup.Group, error) {
	return scanGroupRow(row, false)
}

func scanGroupRow(row rowScanner, hasMemberCount bool) (*domainGroup.Group, error) {
	var (
		id, name, visibility, joinPolicy, createdBy string
		description                                 *string
		isDefault                                   bool
		createdAt, updatedAt                        time.Time
		memberCount                                 int
	)

	dests := []any{&id, &name, &description, &visibility, &joinPolicy, &isDefault, &createdBy, &createdAt, &updatedAt}
	if hasMemberCount {
		dests = append(dests, &memberCount)
	}
	if err := row.Scan(dests...); err != nil {
		return nil, err
	}

	return domainGroup.RestoreGroup(
		id,
		domainGroup.RestoreGroupName(name),
		description,
		domainGroup.RestoreVisibility(visibility),
		domainGroup.RestoreJoinPolicy(joinPolicy),
		isDefault,
		shared.RestoreUserID(createdBy),
		createdAt,
		updatedAt,
	), nil
}

func mapSortClause(f domainGroup.ListFilters, viewerArg func() string) string {
	dir := "ASC"
	if f.Order == domainGroup.OrderDesc {
		dir = "DESC"
	}
	switch f.SortBy {
	case domainGroup.SortByCreatedAt:
		return fmt.Sprintf("g.created_at %s, g.name ASC", dir)
	case domainGroup.SortByMemberCount:
		return fmt.Sprintf("mc %s, g.name ASC", dir)
	case domainGroup.SortByJoinedAt:
		return fmt.Sprintf(
			"(SELECT joined_at FROM group_members gm WHERE gm.group_id = g.id AND gm.user_id = %s) %s, g.name ASC",
			viewerArg(), dir,
		)
	default:
		return fmt.Sprintf("g.name %s", dir)
	}
}
