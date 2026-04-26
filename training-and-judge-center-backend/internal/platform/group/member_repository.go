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
	"github.com/training-judge-center/backend/pkg/apperror"
)

type MemberRepository struct {
	db *pgxpool.Pool
}

func NewMemberRepository(db *pgxpool.Pool) *MemberRepository {
	return &MemberRepository{db: db}
}

func (r *MemberRepository) Save(_ context.Context, _ *domainGroup.GroupMember) error {
	panic("not implemented")
}

func (r *MemberRepository) SaveAll(_ context.Context, _ []*domainGroup.GroupMember) error {
	panic("not implemented")
}

func (r *MemberRepository) FindByGroup(_ context.Context, _ string, _ domainGroup.MemberFilters) ([]*domainGroup.GroupMember, int, error) {
	panic("not implemented")
}

func (r *MemberRepository) FindByGroupAndUser(ctx context.Context, groupID string, userID shared.UserID) (*domainGroup.GroupMember, error) {
	const q = `SELECT id, group_id, user_id, member_role, joined_at FROM group_members WHERE group_id = $1 AND user_id = $2`
	m, err := scanMember(r.db.QueryRow(ctx, q, groupID, userID.Value()))
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
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM group_members WHERE group_id = $1 AND member_role = 'LEAD'`, groupID).Scan(&n)
	if err != nil {
		slog.ErrorContext(ctx, "CountLeads failed", "error", err)
		return 0, apperror.NewInternal()
	}
	return n, nil
}

func (r *MemberRepository) CountMembers(ctx context.Context, groupID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM group_members WHERE group_id = $1`, groupID).Scan(&n)
	if err != nil {
		slog.ErrorContext(ctx, "CountMembers failed", "error", err)
		return 0, apperror.NewInternal()
	}
	return n, nil
}

func (r *MemberRepository) Delete(_ context.Context, _ string, _ shared.UserID) error {
	panic("not implemented")
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

	eg.Go(func() error {
		rows, err := r.db.Query(egCtx,
			`SELECT id, group_id, user_id, member_role, joined_at FROM group_members WHERE group_id = ANY($1) AND user_id = $2`,
			groupIDs, viewerID.Value(),
		)
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
		result[gid] = domainGroup.MemberStats{
			Count:      counts[gid],
			Membership: memberships[gid],
		}
	}
	return result, nil
}

func scanMember(row rowScanner) (*domainGroup.GroupMember, error) {
	var id, gid, uid, role string
	var joinedAt time.Time
	if err := row.Scan(&id, &gid, &uid, &role, &joinedAt); err != nil {
		return nil, err
	}
	return domainGroup.RestoreGroupMember(id, gid, shared.RestoreUserID(uid), domainGroup.RestoreMemberRole(role), joinedAt), nil
}

func (r *MemberRepository) ListLeads(ctx context.Context, groupID string) ([]*domainGroup.GroupMember, error) {
	const q = `SELECT id, group_id, user_id, member_role, joined_at FROM group_members WHERE group_id = $1 AND member_role = 'LEAD' ORDER BY joined_at ASC`
	rows, err := r.db.Query(ctx, q, groupID)
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
