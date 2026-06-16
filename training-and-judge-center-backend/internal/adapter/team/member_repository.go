package team

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type MemberRepository struct {
	db *pgxpool.Pool
}

func NewMemberRepository(db *pgxpool.Pool) *MemberRepository {
	return &MemberRepository{db: db}
}

func (r *MemberRepository) Save(ctx context.Context, m *domainTeam.TeamMember) error {
	const q = `INSERT INTO team_members (id, team_id, user_id, joined_at) VALUES ($1, $2, $3, $4)`
	_, err := infraPostgres.GetQuerier(ctx, r.db).Exec(ctx, q,
		m.ID(), m.TeamID(), m.UserID().Value(), m.JoinedAt(),
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to save team member", "error", err)
		return apperror.NewInternal()
	}
	return nil
}

func (r *MemberRepository) FindByTeam(ctx context.Context, teamID string) ([]*domainTeam.TeamMember, error) {
	const q = `SELECT id, team_id, user_id, joined_at FROM team_members WHERE team_id = $1`
	rows, err := infraPostgres.GetQuerier(ctx, r.db).Query(ctx, q, teamID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to query team members", "error", err, "team_id", teamID)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	var members []*domainTeam.TeamMember
	for rows.Next() {
		var id, tmID, userID string
		var joinedAt time.Time
		if err := rows.Scan(&id, &tmID, &userID, &joinedAt); err != nil {
			slog.ErrorContext(ctx, "failed to scan team member", "error", err)
			return nil, apperror.NewInternal()
		}
		members = append(members, domainTeam.RestoreTeamMember(id, tmID, shared.RestoreUserID(userID), joinedAt))
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "team members rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return members, nil
}
