package material

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GroupProvider struct {
	db *pgxpool.Pool
}

func NewGroupProvider(db *pgxpool.Pool) *GroupProvider {
	return &GroupProvider{db: db}
}

func (p *GroupProvider) Exists(ctx context.Context, groupID string) (bool, error) {
	var exists bool
	err := p.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM groups WHERE id = $1)`, groupID).Scan(&exists)
	if err != nil {
		slog.ErrorContext(ctx, "GroupProvider.Exists failed", "error", err, "group_id", groupID)
		return false, apperror.NewInternal()
	}
	return exists, nil
}

func (p *GroupProvider) FindVisibility(ctx context.Context, groupID string) (appMaterial.GroupVisibility, bool, error) {
	var visibility string
	err := p.db.QueryRow(ctx, `SELECT visibility FROM groups WHERE id = $1`, groupID).Scan(&visibility)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		slog.ErrorContext(ctx, "GroupProvider.FindVisibility failed", "error", err, "group_id", groupID)
		return "", false, apperror.NewInternal()
	}
	return appMaterial.GroupVisibility(visibility), true, nil
}

const memberRoleLead = "LEAD"

type GroupMemberProvider struct {
	db *pgxpool.Pool
}

func NewGroupMemberProvider(db *pgxpool.Pool) *GroupMemberProvider {
	return &GroupMemberProvider{db: db}
}

func (p *GroupMemberProvider) IsLeadOfGroup(ctx context.Context, userID, groupID string) (bool, error) {
	var role string
	err := p.db.QueryRow(ctx,
		`SELECT member_role FROM group_members WHERE group_id = $1 AND user_id = $2`,
		groupID, userID,
	).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		slog.ErrorContext(ctx, "GroupMemberProvider.IsLeadOfGroup failed", "error", err, "group_id", groupID, "user_id", userID)
		return false, apperror.NewInternal()
	}
	if role != "" && role != memberRoleLead {
		slog.WarnContext(ctx, "unrecognised group member role", "role", role, "group_id", groupID, "user_id", userID)
	}
	return role == memberRoleLead, nil
}

func (p *GroupMemberProvider) IsMemberOfGroup(ctx context.Context, userID, groupID string) (bool, error) {
	var exists bool
	err := p.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)`,
		groupID, userID,
	).Scan(&exists)
	if err != nil {
		slog.ErrorContext(ctx, "GroupMemberProvider.IsMemberOfGroup failed", "error", err, "group_id", groupID, "user_id", userID)
		return false, apperror.NewInternal()
	}
	return exists, nil
}
