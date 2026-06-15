package group

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
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

const memberCols = `id, group_id, user_id, member_role, joined_at, added_by, join_method`

type MemberRepository struct {
	db infraPostgres.Querier
}

func NewMemberRepository(db *pgxpool.Pool) *MemberRepository {
	return &MemberRepository{db: db}
}

func (r *MemberRepository) Save(ctx context.Context, m *domainGroup.GroupMember) error {
	const q = `
		INSERT INTO group_members (id, group_id, user_id, member_role, joined_at, added_by, join_method)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	var addedByVal *string
	if ab := m.AddedBy(); ab != nil {
		v := ab.Value()
		addedByVal = &v
	}
	db := infraPostgres.GetQuerier(ctx, r.db)
	_, err := db.Exec(ctx, q,
		m.ID(), m.GroupID(), m.UserID().Value(), m.Role().String(), m.JoinedAt(),
		addedByVal, m.JoinMethod().String(),
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == infraPostgres.UniqueViolation && pgErr.ConstraintName == "group_members_group_id_user_id_key" {
			slog.WarnContext(ctx, "MemberRepository.Save: unique constraint hit (possible TOCTOU race)",
				"group_id", m.GroupID(), "user_id", m.UserID().Value(), "constraint", pgErr.ConstraintName)
			return apperror.NewConflict(domainGroup.ErrCodeAlreadyMember, "You are already a member of this group")
		}
		slog.ErrorContext(ctx, "MemberRepository.Save failed", "error", err)
		return apperror.NewInternal()
	}
	return nil
}

func (r *MemberRepository) Update(ctx context.Context, m *domainGroup.GroupMember) error {
	const q = `UPDATE group_members SET member_role = $1 WHERE group_id = $2 AND user_id = $3`
	db := infraPostgres.GetQuerier(ctx, r.db)
	tag, err := db.Exec(ctx, q, m.Role().String(), m.GroupID(), m.UserID().Value())
	if err != nil {
		slog.ErrorContext(ctx, "MemberRepository.Update failed", "group_id", m.GroupID(), "user_id", m.UserID().Value(), "error", err)
		return apperror.NewInternal()
	}
	if tag.RowsAffected() == 0 {
		return apperror.NewNotFound(domainGroup.ErrCodeNotAMember, "user is not a member of this group")
	}
	return nil
}

func (r *MemberRepository) SaveAll(_ context.Context, _ []*domainGroup.GroupMember) error {
	panic("not implemented")
}

func (r *MemberRepository) FindByGroup(ctx context.Context, groupID string, filters domainGroup.MemberFilters) ([]*domainGroup.GroupMember, int, error) {
	db := infraPostgres.GetQuerier(ctx, r.db)
	where := `WHERE group_id = $1`
	args := []any{groupID}
	argIdx := 2

	if filters.Role != nil {
		where += ` AND member_role = $` + strconv.Itoa(argIdx)
		args = append(args, filters.Role.String())
		argIdx++
	}

	var total int
	countQ := `SELECT COUNT(*) FROM group_members ` + where
	if err := db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		slog.ErrorContext(ctx, "FindByGroup count failed", "group_id", groupID, "error", err)
		return nil, 0, apperror.NewInternal()
	}

	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := (filters.Page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	q := `SELECT ` + memberCols + ` FROM group_members ` + where +
		` ORDER BY joined_at ASC LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1)
	args = append(args, limit, offset)

	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		slog.ErrorContext(ctx, "FindByGroup query failed", "group_id", groupID, "error", err)
		return nil, 0, apperror.NewInternal()
	}
	defer rows.Close()

	var out []*domainGroup.GroupMember
	for rows.Next() {
		m, err := scanMember(rows)
		if err != nil {
			slog.ErrorContext(ctx, "FindByGroup scan failed", "group_id", groupID, "error", err)
			return nil, 0, apperror.NewInternal()
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "FindByGroup rows error", "group_id", groupID, "error", err)
		return nil, 0, apperror.NewInternal()
	}
	return out, total, nil
}

// FindByGroupAndUser returns (nil, nil) when the user is not a member of the group.
// Callers must check for a nil member before accessing its fields.
func (r *MemberRepository) FindByGroupAndUser(ctx context.Context, groupID string, userID shared.UserID) (*domainGroup.GroupMember, error) {
	db := infraPostgres.GetQuerier(ctx, r.db)
	q := `SELECT ` + memberCols + ` FROM group_members WHERE group_id = $1 AND user_id = $2`
	m, err := scanMember(db.QueryRow(ctx, q, groupID, userID.Value()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "FindByGroupAndUser failed", "error", err)
		return nil, apperror.NewInternal()
	}
	return m, nil
}

func (r *MemberRepository) CountLeads(ctx context.Context, groupID string) (int, error) {
	db := infraPostgres.GetQuerier(ctx, r.db)
	var n int
	err := db.QueryRow(ctx, `SELECT COUNT(*) FROM group_members WHERE group_id = $1 AND member_role = 'LEAD'`, groupID).Scan(&n)
	if err != nil {
		slog.ErrorContext(ctx, "CountLeads failed", "error", err)
		return 0, apperror.NewInternal()
	}
	return n, nil
}

func (r *MemberRepository) CountMembers(ctx context.Context, groupID string) (int, error) {
	db := infraPostgres.GetQuerier(ctx, r.db)
	var n int
	err := db.QueryRow(ctx, `SELECT COUNT(*) FROM group_members WHERE group_id = $1`, groupID).Scan(&n)
	if err != nil {
		slog.ErrorContext(ctx, "CountMembers failed", "error", err)
		return 0, apperror.NewInternal()
	}
	return n, nil
}

func (r *MemberRepository) Delete(ctx context.Context, groupID string, userID shared.UserID) error {
	const q = `DELETE FROM group_members WHERE group_id = $1 AND user_id = $2`
	db := infraPostgres.GetQuerier(ctx, r.db)
	tag, err := db.Exec(ctx, q, groupID, userID.Value())
	if err != nil {
		slog.ErrorContext(ctx, "MemberRepository.Delete failed", "group_id", groupID, "user_id", userID.Value(), "error", err)
		return apperror.NewInternal()
	}
	if tag.RowsAffected() == 0 {
		return apperror.NewNotFound(domainGroup.ErrCodeNotAMember, "user is not a member of this group")
	}
	return nil
}

func (r *MemberRepository) BulkStats(ctx context.Context, groupIDs []string, viewerID shared.UserID) (map[string]domainGroup.MemberStats, error) {
	if len(groupIDs) == 0 {
		return map[string]domainGroup.MemberStats{}, nil
	}

	db := infraPostgres.GetQuerier(ctx, r.db)
	counts := map[string]int{}
	memberships := map[string]*domainGroup.GroupMember{}

	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error { // writes only to counts — no other goroutine touches counts
		rows, err := db.Query(egCtx,
			`SELECT group_id, COUNT(*) FROM group_members WHERE group_id = ANY($1) GROUP BY group_id`,
			groupIDs,
		)
		if err != nil {
			slog.ErrorContext(egCtx, "BulkStats count failed", "error", err)
			return apperror.NewInternal()
		}
		defer rows.Close()
		for rows.Next() {
			var gid string
			var n int
			if err := rows.Scan(&gid, &n); err != nil {
				slog.ErrorContext(egCtx, "BulkStats count scan failed", "error", err)
				return apperror.NewInternal()
			}
			counts[gid] = n
		}
		if err := rows.Err(); err != nil {
			slog.ErrorContext(egCtx, "BulkStats count rows error", "error", err)
			return apperror.NewInternal()
		}
		return nil
	})

	eg.Go(func() error { // writes only to memberships — no other goroutine touches memberships
		q := `SELECT ` + memberCols + ` FROM group_members WHERE group_id = ANY($1) AND user_id = $2`
		rows, err := db.Query(egCtx, q, groupIDs, viewerID.Value())
		if err != nil {
			slog.ErrorContext(egCtx, "BulkStats memberships failed", "error", err)
			return apperror.NewInternal()
		}
		defer rows.Close()
		for rows.Next() {
			m, err := scanMember(rows)
			if err != nil {
				slog.ErrorContext(egCtx, "BulkStats membership scan failed", "error", err)
				return apperror.NewInternal()
			}
			memberships[m.GroupID()] = m
		}
		if err := rows.Err(); err != nil {
			slog.ErrorContext(egCtx, "BulkStats membership rows error", "error", err)
			return apperror.NewInternal()
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	result := make(map[string]domainGroup.MemberStats, len(groupIDs))
	for _, gid := range groupIDs {
		s := domainGroup.MemberStats{Count: counts[gid]}
		if m := memberships[gid]; m != nil {
			s.IsMember = true
			s.Role = m.Role()
			s.JoinedAt = m.JoinedAt()
		}
		result[gid] = s
	}
	return result, nil
}

func scanMember(row rowScanner) (*domainGroup.GroupMember, error) {
	var id, gid, uid, role, joinMethod string
	var joinedAt time.Time
	var addedByRaw *string
	if err := row.Scan(&id, &gid, &uid, &role, &joinedAt, &addedByRaw, &joinMethod); err != nil {
		return nil, err
	}
	var addedBy *shared.UserID
	if addedByRaw != nil {
		v := shared.RestoreUserID(*addedByRaw)
		addedBy = &v
	}
	return domainGroup.RestoreGroupMember(
		id, gid, shared.RestoreUserID(uid),
		domainGroup.RestoreMemberRole(role), joinedAt,
		addedBy, domainGroup.RestoreJoinMethod(joinMethod),
	), nil
}

func (r *MemberRepository) ListLeads(ctx context.Context, groupID string) ([]*domainGroup.GroupMember, error) {
	db := infraPostgres.GetQuerier(ctx, r.db)
	q := `SELECT ` + memberCols + ` FROM group_members WHERE group_id = $1 AND member_role = 'LEAD' ORDER BY joined_at ASC`
	rows, err := db.Query(ctx, q, groupID)
	if err != nil {
		slog.ErrorContext(ctx, "ListLeads failed", "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	var out []*domainGroup.GroupMember
	for rows.Next() {
		m, err := scanMember(rows)
		if err != nil {
			slog.ErrorContext(ctx, "database error scanning group member row in ListLeads", "group_id", groupID, "error", err)
			return nil, apperror.NewInternal()
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "database error iterating group member rows in ListLeads", "group_id", groupID, "error", err)
		return nil, apperror.NewInternal()
	}
	return out, nil
}
