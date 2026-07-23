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

type InvitationRepository struct {
	db infraPostgres.Querier
}

func NewInvitationRepository(db infraPostgres.Querier) *InvitationRepository {
	return &InvitationRepository{db: db}
}

func (r *InvitationRepository) Save(ctx context.Context, inv *domainTeam.TeamInvitation) error {
	const q = `INSERT INTO team_invitations (id, team_id, invitee_id, invited_by, created_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := infraPostgres.GetQuerier(ctx, r.db).Exec(ctx, q,
		inv.ID(), inv.TeamID(), inv.InviteeID().Value(), inv.InvitedBy().Value(), inv.CreatedAt(),
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == infraPostgres.UniqueViolation && pgErr.ConstraintName == "team_invitations_team_invitee_key" {
			slog.WarnContext(ctx, "InvitationRepository.Save: duplicate invitation (possible TOCTOU race)",
				"team_id", inv.TeamID(), "invitee_id", inv.InviteeID().Value())
			return apperror.NewConflict(domainTeam.ErrCodeAlreadyInvited, "User already has a pending invitation to this team")
		}
		slog.ErrorContext(ctx, "failed to save team invitation", "error", err)
		return apperror.NewInternal()
	}
	return nil
}

func (r *InvitationRepository) FindByID(ctx context.Context, id string) (*domainTeam.TeamInvitation, error) {
	const q = `SELECT id, team_id, invitee_id, invited_by, created_at FROM team_invitations WHERE id = $1`
	var invID, teamID, inviteeID, invitedBy string
	var createdAt time.Time
	err := infraPostgres.GetQuerier(ctx, r.db).QueryRow(ctx, q, id).Scan(&invID, &teamID, &inviteeID, &invitedBy, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound(domainTeam.ErrCodeInvitationNotFound, "Invitation not found")
		}
		slog.ErrorContext(ctx, "failed to find invitation by id", "error", err, "invitation_id", id)
		return nil, apperror.NewInternal()
	}
	return domainTeam.RestoreTeamInvitation(invID, teamID, shared.RestoreUserID(inviteeID), shared.RestoreUserID(invitedBy), createdAt), nil
}

func (r *InvitationRepository) FindByTeam(ctx context.Context, teamID string) ([]*domainTeam.TeamInvitation, error) {
	const q = `SELECT id, team_id, invitee_id, invited_by, created_at FROM team_invitations WHERE team_id = $1 ORDER BY created_at DESC`
	rows, err := infraPostgres.GetQuerier(ctx, r.db).Query(ctx, q, teamID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to query invitations by team", "error", err, "team_id", teamID)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	invitations := make([]*domainTeam.TeamInvitation, 0)
	for rows.Next() {
		var id, tID, iID, invBy string
		var createdAt time.Time
		if err := rows.Scan(&id, &tID, &iID, &invBy, &createdAt); err != nil {
			slog.ErrorContext(ctx, "failed to scan invitation row", "error", err)
			return nil, apperror.NewInternal()
		}
		invitations = append(invitations, domainTeam.RestoreTeamInvitation(id, tID, shared.RestoreUserID(iID), shared.RestoreUserID(invBy), createdAt))
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "invitations by team rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return invitations, nil
}

func (r *InvitationRepository) FindByTeamAndInvitee(ctx context.Context, teamID string, inviteeID shared.UserID) (*domainTeam.TeamInvitation, error) {
	const q = `SELECT id, team_id, invitee_id, invited_by, created_at FROM team_invitations WHERE team_id = $1 AND invitee_id = $2`
	var invID, tID, iID, invBy string
	var createdAt time.Time
	err := infraPostgres.GetQuerier(ctx, r.db).QueryRow(ctx, q, teamID, inviteeID.Value()).Scan(&invID, &tID, &iID, &invBy, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound(domainTeam.ErrCodeInvitationNotFound, "Invitation not found")
		}
		slog.ErrorContext(ctx, "failed to find invitation by team and invitee", "error", err)
		return nil, apperror.NewInternal()
	}
	return domainTeam.RestoreTeamInvitation(invID, tID, shared.RestoreUserID(iID), shared.RestoreUserID(invBy), createdAt), nil
}

func (r *InvitationRepository) FindByInvitee(ctx context.Context, inviteeID shared.UserID) ([]*domainTeam.TeamInvitation, error) {
	const q = `SELECT id, team_id, invitee_id, invited_by, created_at FROM team_invitations WHERE invitee_id = $1 ORDER BY created_at DESC`
	rows, err := infraPostgres.GetQuerier(ctx, r.db).Query(ctx, q, inviteeID.Value())
	if err != nil {
		slog.ErrorContext(ctx, "failed to query invitations by invitee", "error", err, "invitee_id", inviteeID.Value())
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	invitations := make([]*domainTeam.TeamInvitation, 0)
	for rows.Next() {
		var id, teamID, iID, invBy string
		var createdAt time.Time
		if err := rows.Scan(&id, &teamID, &iID, &invBy, &createdAt); err != nil {
			slog.ErrorContext(ctx, "failed to scan invitation row", "error", err)
			return nil, apperror.NewInternal()
		}
		invitations = append(invitations, domainTeam.RestoreTeamInvitation(id, teamID, shared.RestoreUserID(iID), shared.RestoreUserID(invBy), createdAt))
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "invitations by invitee rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return invitations, nil
}

func (r *InvitationRepository) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM team_invitations WHERE id = $1`
	tag, err := infraPostgres.GetQuerier(ctx, r.db).Exec(ctx, q, id)
	if err != nil {
		slog.ErrorContext(ctx, "failed to delete team invitation", "error", err, "invitation_id", id)
		return apperror.NewInternal()
	}
	if tag.RowsAffected() == 0 {
		return apperror.NewNotFound(domainTeam.ErrCodeInvitationNotFound, "Invitation not found")
	}
	return nil
}
