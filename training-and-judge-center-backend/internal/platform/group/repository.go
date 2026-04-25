package group

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

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GroupRepository struct {
	db *pgxpool.Pool
}

func NewGroupRepository(db *pgxpool.Pool) *GroupRepository {
	return &GroupRepository{db: db}
}

func (r *GroupRepository) Save(_ context.Context, _ *domainGroup.Group) error {
	panic("not implemented")
}

func (r *GroupRepository) FindByID(ctx context.Context, id string) (*domainGroup.Group, error) {
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

func (r *GroupRepository) ExistsByName(_ context.Context, _ domainGroup.GroupName) (bool, error) {
	panic("not implemented")
}

func (r *GroupRepository) FindDefault(_ context.Context) (*domainGroup.Group, error) {
	panic("not implemented")
}

func (r *GroupRepository) List(ctx context.Context, filters domainGroup.ListFilters) ([]*domainGroup.Group, int, error) {
	var conds []string
	var args []any
	idx := 1
	nextArg := func(v any) string {
		args = append(args, v)
		s := fmt.Sprintf("$%d", idx)
		idx++
		return s
	}

	var viewerArgStr string
	viewerArg := func() string {
		if viewerArgStr == "" {
			viewerArgStr = nextArg(filters.ViewerID.Value())
		}
		return viewerArgStr
	}

	if filters.OnlyMyGroups {
		if filters.RoleFilter != nil {
			roleArg := nextArg(string(*filters.RoleFilter))
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
			return err
		}
		defer rows.Close()
		for rows.Next() {
			g, err := scanGroupRow(rows, memberCountSelect != "")
			if err != nil {
				return err
			}
			result = append(result, g)
		}
		return rows.Err()
	})

	eg.Go(func() error {
		return r.db.QueryRow(gCtx, countQuery, countArgs...).Scan(&total)
	})

	if err := eg.Wait(); err != nil {
		slog.ErrorContext(ctx, "List groups failed", "error", err)
		return nil, 0, apperror.NewInternal()
	}
	if result == nil {
		result = []*domainGroup.Group{}
	}
	return result, total, nil
}

func (r *GroupRepository) Delete(_ context.Context, _ string) error {
	panic("not implemented")
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
