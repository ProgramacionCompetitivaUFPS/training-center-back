package group

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	infraPostgres "github.com/training-judge-center/backend/internal/infrastructure/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type MemberRepository struct {
	db *pgxpool.Pool
}

func NewMemberRepository(db *pgxpool.Pool) *MemberRepository {
	return &MemberRepository{db: db}
}

func (r *MemberRepository) Save(ctx context.Context, m *domainGroup.GroupMember) error {
	q := infraPostgres.GetQuerier(ctx, r.db)
	var addedByID *string
	if ab := m.AddedBy(); ab != nil {
		v := ab.Value()
		addedByID = &v
	}
	const sql = `
		INSERT INTO group_members (id, group_id, user_id, member_role, joined_at, added_by, join_method)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET member_role = EXCLUDED.member_role`
	if _, err := q.Exec(ctx, sql,
		m.ID(), m.GroupID(), m.UserID().Value(),
		string(m.Role()), m.JoinedAt(), addedByID, string(m.JoinMethod()),
	); err != nil {
		slog.ErrorContext(ctx, "MemberRepository.Save failed", "error", err)
		return apperror.NewInternal()
	}
	return nil
}

func (r *MemberRepository) SaveAll(ctx context.Context, members []*domainGroup.GroupMember) error {
	if len(members) == 0 {
		return nil
	}
	q := infraPostgres.GetQuerier(ctx, r.db)
	for _, m := range members {
		var addedByID *string
		if ab := m.AddedBy(); ab != nil {
			v := ab.Value()
			addedByID = &v
		}
		const sql = `
			INSERT INTO group_members (id, group_id, user_id, member_role, joined_at, added_by, join_method)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (id) DO NOTHING`
		if _, err := q.Exec(ctx, sql,
			m.ID(), m.GroupID(), m.UserID().Value(),
			string(m.Role()), m.JoinedAt(), addedByID, string(m.JoinMethod()),
		); err != nil {
			slog.ErrorContext(ctx, "MemberRepository.SaveAll failed", "error", err)
			return apperror.NewInternal()
		}
	}
	return nil
}

const selectMemberCols = `id, group_id, user_id, member_role, joined_at, added_by, join_method, removed_at`

func (r *MemberRepository) FindByGroupAndUser(ctx context.Context, groupID string, userID shared.UserID) (*domainGroup.GroupMember, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)
	sql := `SELECT ` + selectMemberCols + ` FROM group_members WHERE group_id = $1 AND user_id = $2 AND removed_at IS NULL`
	m, err := scanMember(q.QueryRow(ctx, sql, groupID, userID.Value()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "FindByGroupAndUser failed", "error", err)
		return nil, apperror.NewInternal()
	}
	return m, nil
}

func (r *MemberRepository) FindByGroup(_ context.Context, _ string, _ domainGroup.MemberFilters) ([]*domainGroup.GroupMember, int, error) {
	panic("FindByGroup not implemented")
}

func (r *MemberRepository) Delete(_ context.Context, _ string, _ shared.UserID) error {
	panic("Delete not implemented")
}

func (r *MemberRepository) CountLeads(ctx context.Context, groupID string) (int, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)
	var n int
	err := q.QueryRow(ctx, `SELECT COUNT(*) FROM group_members WHERE group_id = $1 AND member_role = 'LEAD' AND removed_at IS NULL`, groupID).Scan(&n)
	if err != nil {
		slog.ErrorContext(ctx, "CountLeads failed", "error", err)
		return 0, apperror.NewInternal()
	}
	return n, nil
}

func (r *MemberRepository) CountMembers(ctx context.Context, groupID string) (int, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)
	var n int
	err := q.QueryRow(ctx, `SELECT COUNT(*) FROM group_members WHERE group_id = $1 AND removed_at IS NULL`, groupID).Scan(&n)
	if err != nil {
		slog.ErrorContext(ctx, "CountMembers failed", "error", err)
		return 0, apperror.NewInternal()
	}
	return n, nil
}

func (r *MemberRepository) ListLeads(ctx context.Context, groupID string) ([]*domainGroup.GroupMember, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)
	sql := `SELECT ` + selectMemberCols + ` FROM group_members WHERE group_id = $1 AND member_role = 'LEAD' AND removed_at IS NULL ORDER BY joined_at ASC`
	rows, err := q.Query(ctx, sql, groupID)
	if err != nil {
		slog.ErrorContext(ctx, "ListLeads failed", "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	var out []*domainGroup.GroupMember
	for rows.Next() {
		m, err := scanMember(rows)
		if err != nil {
			return nil, apperror.NewInternal()
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.NewInternal()
	}
	return out, nil
}

func (r *MemberRepository) BulkStats(ctx context.Context, groupIDs []string, viewerID shared.UserID) (map[string]domainGroup.MemberStats, error) {
	if len(groupIDs) == 0 {
		return map[string]domainGroup.MemberStats{}, nil
	}

	counts := map[string]int{}
	memberships := map[string]*domainGroup.GroupMember{}

	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		rows, err := r.db.Query(egCtx,
			`SELECT group_id, COUNT(*) FROM group_members WHERE group_id = ANY($1) AND removed_at IS NULL GROUP BY group_id`,
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
				return apperror.NewInternal()
			}
			counts[gid] = n
		}
		return rows.Err()
	})

	eg.Go(func() error {
		sql := `SELECT ` + selectMemberCols + ` FROM group_members WHERE group_id = ANY($1) AND user_id = $2 AND removed_at IS NULL`
		rows, err := r.db.Query(egCtx, sql, groupIDs, viewerID.Value())
		if err != nil {
			slog.ErrorContext(egCtx, "BulkStats memberships failed", "error", err)
			return apperror.NewInternal()
		}
		defer rows.Close()
		for rows.Next() {
			m, err := scanMember(rows)
			if err != nil {
				return apperror.NewInternal()
			}
			memberships[m.GroupID()] = m
		}
		return rows.Err()
	})

	if err := eg.Wait(); err != nil {
		return nil, apperror.NewInternal()
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
	var addedByStr *string
	var removedAt *time.Time
	if err := row.Scan(&id, &gid, &uid, &role, &joinedAt, &addedByStr, &joinMethod, &removedAt); err != nil {
		return nil, err
	}
	var addedBy *shared.UserID
	if addedByStr != nil {
		v := shared.RestoreUserID(*addedByStr)
		addedBy = &v
	}
	return domainGroup.RestoreGroupMember(
		id, gid,
		shared.RestoreUserID(uid),
		domainGroup.RestoreMemberRole(role),
		joinedAt,
		addedBy,
		domainGroup.RestoreJoinMethod(joinMethod),
		removedAt,
	), nil
}
