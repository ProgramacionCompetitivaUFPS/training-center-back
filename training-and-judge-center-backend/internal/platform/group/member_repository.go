package group

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

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

// Save, SaveAll, Delete son placeholders — G-2/G-4/G-7 los implementan.

func (r *MemberRepository) Save(ctx context.Context, m *domainGroup.GroupMember) error {
	return apperror.NewInternal()
}

func (r *MemberRepository) SaveAll(ctx context.Context, members []*domainGroup.GroupMember) error {
	return apperror.NewInternal()
}

func (r *MemberRepository) Delete(ctx context.Context, groupID string, userID shared.UserID) error {
	return apperror.NewInternal()
}

func (r *MemberRepository) FindByGroupAndUser(ctx context.Context, groupID string, userID shared.UserID) (*domainGroup.GroupMember, error) {
	const q = `SELECT id, group_id, user_id, member_role, joined_at FROM group_members WHERE group_id = $1 AND user_id = $2`
	row := r.db.QueryRow(ctx, q, groupID, userID.Value())
	var (
		id, gid, uid, role string
		joinedAt           time.Time
	)
	if err := row.Scan(&id, &gid, &uid, &role, &joinedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "FindByGroupAndUser failed", "error", err)
		return nil, apperror.NewInternal()
	}
	mr, err := domainGroup.NewMemberRole(role)
	if err != nil {
		return nil, apperror.NewInternal()
	}
	return domainGroup.RestoreGroupMember(id, gid, shared.RestoreUserID(uid), mr, joinedAt), nil
}

// FindByGroup es placeholder — G-7 lo implementa.
func (r *MemberRepository) FindByGroup(ctx context.Context, groupID string, filters domainGroup.MemberFilters) ([]*domainGroup.GroupMember, int, error) {
	return nil, 0, apperror.NewInternal()
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
		var (
			id, gid, uid, role string
			joinedAt           time.Time
		)
		if err := rows.Scan(&id, &gid, &uid, &role, &joinedAt); err != nil {
			return nil, apperror.NewInternal()
		}
		mr, err := domainGroup.NewMemberRole(role)
		if err != nil {
			return nil, apperror.NewInternal()
		}
		out = append(out, domainGroup.RestoreGroupMember(id, gid, shared.RestoreUserID(uid), mr, joinedAt))
	}
	if err := rows.Err(); err != nil {
		return nil, apperror.NewInternal()
	}
	return out, nil
}
