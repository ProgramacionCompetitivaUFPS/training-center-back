package team

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type MemberRepository struct {
	db infraPostgres.Querier
}

func NewMemberRepository(db infraPostgres.Querier) *MemberRepository {
	return &MemberRepository{db: db}
}

func (r *MemberRepository) Save(ctx context.Context, m *domainTeam.TeamMember) error {
	const q = `INSERT INTO team_members (id, team_id, user_id, joined_at) VALUES ($1, $2, $3, $4)`
	_, err := infraPostgres.GetQuerier(ctx, r.db).Exec(ctx, q,
		m.ID(), m.TeamID(), m.UserID().Value(), m.JoinedAt(),
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == infraPostgres.UniqueViolation && pgErr.ConstraintName == "team_members_team_user_key" {
			slog.WarnContext(ctx, "MemberRepository.Save: unique constraint hit (possible TOCTOU race)",
				"team_id", m.TeamID(), "user_id", m.UserID().Value(), "constraint", pgErr.ConstraintName)
			return apperror.NewConflict(domainTeam.ErrCodeMemberAlreadyOnTeam, "User is already a member of this team")
		}
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

	members := make([]*domainTeam.TeamMember, 0)
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

func (r *MemberRepository) FindByUser(ctx context.Context, userID shared.UserID) ([]*domainTeam.TeamMember, error) {
	const q = `SELECT id, team_id, user_id, joined_at FROM team_members WHERE user_id = $1 ORDER BY joined_at DESC`
	rows, err := infraPostgres.GetQuerier(ctx, r.db).Query(ctx, q, userID.Value())
	if err != nil {
		slog.ErrorContext(ctx, "failed to query team members by user", "error", err, "user_id", userID.Value())
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	members := make([]*domainTeam.TeamMember, 0)
	for rows.Next() {
		var id, tmID, uid string
		var joinedAt time.Time
		if err := rows.Scan(&id, &tmID, &uid, &joinedAt); err != nil {
			slog.ErrorContext(ctx, "failed to scan team member row", "error", err)
			return nil, apperror.NewInternal()
		}
		members = append(members, domainTeam.RestoreTeamMember(id, tmID, shared.RestoreUserID(uid), joinedAt))
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "team members by user rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return members, nil
}

func (r *MemberRepository) FindByTeamAndUser(ctx context.Context, teamID string, userID shared.UserID) (*domainTeam.TeamMember, error) {
	const q = `SELECT id, team_id, user_id, joined_at FROM team_members WHERE team_id = $1 AND user_id = $2`
	var id, tmID, uid string
	var joinedAt time.Time
	err := infraPostgres.GetQuerier(ctx, r.db).QueryRow(ctx, q, teamID, userID.Value()).Scan(&id, &tmID, &uid, &joinedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound(domainTeam.ErrCodeNotTeamMember, "User is not a member of this team")
		}
		slog.ErrorContext(ctx, "failed to find team member by team and user", "error", err)
		return nil, apperror.NewInternal()
	}
	return domainTeam.RestoreTeamMember(id, tmID, shared.RestoreUserID(uid), joinedAt), nil
}

func (r *MemberRepository) DeleteByTeamAndUser(ctx context.Context, teamID string, userID shared.UserID) error {
	const q = `DELETE FROM team_members WHERE team_id = $1 AND user_id = $2`
	_, err := infraPostgres.GetQuerier(ctx, r.db).Exec(ctx, q, teamID, userID.Value())
	if err != nil {
		slog.ErrorContext(ctx, "failed to delete team member", "error", err, "team_id", teamID, "user_id", userID.Value())
		return apperror.NewInternal()
	}
	return nil
}

func (r *MemberRepository) BulkCountByTeams(ctx context.Context, teamIDs []string) (map[string]int, error) {
	if len(teamIDs) == 0 {
		return map[string]int{}, nil
	}

	const q = `SELECT team_id, COUNT(*) FROM team_members WHERE team_id = ANY($1) GROUP BY team_id`
	rows, err := infraPostgres.GetQuerier(ctx, r.db).Query(ctx, q, teamIDs)
	if err != nil {
		slog.ErrorContext(ctx, "failed to bulk count team members", "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	counts := make(map[string]int, len(teamIDs))
	for rows.Next() {
		var teamID string
		var count int
		if err := rows.Scan(&teamID, &count); err != nil {
			slog.ErrorContext(ctx, "failed to scan team member count", "error", err)
			return nil, apperror.NewInternal()
		}
		counts[teamID] = count
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "bulk count rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return counts, nil
}
