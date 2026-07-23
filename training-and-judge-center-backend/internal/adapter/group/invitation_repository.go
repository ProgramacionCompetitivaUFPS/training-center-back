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

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const invitationCols = `id, group_id, invitee_id, invited_by, status, expires_at, created_at`

type InvitationRepository struct {
	db infraPostgres.Querier
}

func NewInvitationRepository(db *pgxpool.Pool) *InvitationRepository {
	return &InvitationRepository{db: db}
}

func (r *InvitationRepository) Save(ctx context.Context, inv *domainGroup.GroupInvitation) error {
	const q = `
		INSERT INTO group_invitations (id, group_id, invitee_id, invited_by, status, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	var inviteeVal *string
	if id := inv.InviteeID(); id != nil {
		v := id.Value()
		inviteeVal = &v
	}
	db := infraPostgres.GetQuerier(ctx, r.db)
	_, err := db.Exec(ctx, q,
		inv.ID(), inv.GroupID(), inviteeVal, inv.InvitedBy().Value(),
		inv.Status().String(), inv.ExpiresAt(), inv.CreatedAt(),
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == infraPostgres.UniqueViolation && pgErr.ConstraintName == "idx_group_invitations_pending_unique" {
			slog.WarnContext(ctx, "InvitationRepository.Save: pending invitation already exists (possible TOCTOU race)",
				"group_id", inv.GroupID())
			return apperror.NewConflict(domainGroup.ErrCodeInvitationAlreadyProcessed, "a pending invitation already exists for this invitee")
		}
		slog.ErrorContext(ctx, "InvitationRepository.Save failed", "error", err)
		return apperror.NewInternal()
	}
	return nil
}

func (r *InvitationRepository) FindByID(ctx context.Context, id string) (*domainGroup.GroupInvitation, error) {
	db := infraPostgres.GetQuerier(ctx, r.db)
	q := `SELECT ` + invitationCols + ` FROM group_invitations WHERE id = $1`
	inv, err := scanInvitation(db.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound(domainGroup.ErrCodeInvitationNotFound, "invitation not found")
		}
		slog.ErrorContext(ctx, "InvitationRepository.FindByID failed", "error", err)
		return nil, apperror.NewInternal()
	}
	return inv, nil
}

// FindPendingByGroupAndInvitee returns (nil, nil) when no matching PENDING
// invitation exists. inviteeID == nil looks up the general (link-only)
// invitation for the group instead of a per-invitee one.
func (r *InvitationRepository) FindPendingByGroupAndInvitee(ctx context.Context, groupID string, inviteeID *shared.UserID) (*domainGroup.GroupInvitation, error) {
	db := infraPostgres.GetQuerier(ctx, r.db)

	var row pgx.Row
	if inviteeID == nil {
		q := `SELECT ` + invitationCols + ` FROM group_invitations WHERE group_id = $1 AND invitee_id IS NULL AND status = 'PENDING' LIMIT 1`
		row = db.QueryRow(ctx, q, groupID)
	} else {
		q := `SELECT ` + invitationCols + ` FROM group_invitations WHERE group_id = $1 AND invitee_id = $2 AND status = 'PENDING' LIMIT 1`
		row = db.QueryRow(ctx, q, groupID, inviteeID.Value())
	}

	inv, err := scanInvitation(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "InvitationRepository.FindPendingByGroupAndInvitee failed", "error", err)
		return nil, apperror.NewInternal()
	}
	return inv, nil
}

func (r *InvitationRepository) FindByGroup(ctx context.Context, groupID string, filters domainGroup.InvitationFilters) ([]*domainGroup.GroupInvitation, int, error) {
	db := infraPostgres.GetQuerier(ctx, r.db)
	where := `WHERE group_id = $1`
	args := []any{groupID}
	argIdx := 2

	if filters.Status != nil {
		where += ` AND status = $` + strconv.Itoa(argIdx)
		args = append(args, filters.Status.String())
		argIdx++
	}

	var total int
	countQ := `SELECT COUNT(*) FROM group_invitations ` + where
	if err := db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		slog.ErrorContext(ctx, "InvitationRepository.FindByGroup count failed", "group_id", groupID, "error", err)
		return nil, 0, apperror.NewInternal()
	}

	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}
	page := filters.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	q := `SELECT ` + invitationCols + ` FROM group_invitations ` + where +
		` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1)
	args = append(args, limit, offset)

	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		slog.ErrorContext(ctx, "InvitationRepository.FindByGroup query failed", "group_id", groupID, "error", err)
		return nil, 0, apperror.NewInternal()
	}
	defer rows.Close()

	var out []*domainGroup.GroupInvitation
	for rows.Next() {
		inv, err := scanInvitation(rows)
		if err != nil {
			slog.ErrorContext(ctx, "InvitationRepository.FindByGroup scan failed", "group_id", groupID, "error", err)
			return nil, 0, apperror.NewInternal()
		}
		out = append(out, inv)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "InvitationRepository.FindByGroup rows error", "group_id", groupID, "error", err)
		return nil, 0, apperror.NewInternal()
	}
	if out == nil {
		out = []*domainGroup.GroupInvitation{}
	}
	return out, total, nil
}

func (r *InvitationRepository) TransitionStatus(ctx context.Context, id string, from, to domainGroup.InvitationStatus) error {
	const q = `UPDATE group_invitations SET status = $1 WHERE id = $2 AND status = $3`
	db := infraPostgres.GetQuerier(ctx, r.db)
	tag, err := db.Exec(ctx, q, to.String(), id, from.String())
	if err != nil {
		slog.ErrorContext(ctx, "InvitationRepository.TransitionStatus failed", "invitation_id", id, "error", err)
		return apperror.NewInternal()
	}
	if tag.RowsAffected() == 0 {
		return apperror.NewConflict(domainGroup.ErrCodeInvitationAlreadyProcessed, "invitation is no longer in the expected state")
	}
	return nil
}

func scanInvitation(row rowScanner) (*domainGroup.GroupInvitation, error) {
	var id, groupID, invitedBy, status string
	var inviteeRaw *string
	var expiresAt, createdAt time.Time
	if err := row.Scan(&id, &groupID, &inviteeRaw, &invitedBy, &status, &expiresAt, &createdAt); err != nil {
		return nil, err
	}
	var inviteeID *shared.UserID
	if inviteeRaw != nil {
		v := shared.RestoreUserID(*inviteeRaw)
		inviteeID = &v
	}
	return domainGroup.RestoreGroupInvitation(
		id, groupID, inviteeID, shared.RestoreUserID(invitedBy),
		domainGroup.RestoreInvitationStatus(status), expiresAt, createdAt,
	), nil
}
