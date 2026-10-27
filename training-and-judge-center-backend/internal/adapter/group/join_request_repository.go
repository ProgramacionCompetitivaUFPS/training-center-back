package group

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type JoinRequestRepository struct {
	db *pgxpool.Pool
}

func NewJoinRequestRepository(db *pgxpool.Pool) *JoinRequestRepository {
	return &JoinRequestRepository{db: db}
}

func (r *JoinRequestRepository) Save(ctx context.Context, req *domainGroup.JoinRequest) error {
	const q = `
		INSERT INTO join_requests (id, group_id, requester_user_id, message, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status
	`
	db := infraPostgres.GetQuerier(ctx, r.db)
	_, err := db.Exec(ctx, q,
		req.ID(),
		req.GroupID(),
		req.RequesterUserID().Value(),
		req.Message(),
		req.Status().String(),
		req.CreatedAt(),
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == infraPostgres.UniqueViolation {
			return apperror.NewConflict(domainGroup.ErrCodeRequestAlreadyPending, "a pending request already exists for this user and group")
		}
		slog.ErrorContext(ctx, "JoinRequestRepository.Save failed", "error", err)
		return apperror.NewInternal()
	}
	return nil
}

func (r *JoinRequestRepository) FindByID(ctx context.Context, id string) (*domainGroup.JoinRequest, error) {
	const q = `
		SELECT id, group_id, requester_user_id, message, status, created_at
		FROM join_requests WHERE id = $1
	`
	req, err := scanJoinRequest(infraPostgres.GetQuerier(ctx, r.db).QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "JoinRequestRepository.FindByID failed", "error", err)
		return nil, apperror.NewInternal()
	}
	return req, nil
}

func (r *JoinRequestRepository) FindByGroupAndUser(ctx context.Context, groupID string, userID shared.UserID) (*domainGroup.JoinRequest, error) {
	// PENDING is sorted first: a user can have at most one pending request
	// (enforced by idx_join_requests_pending), but historical closed requests
	// remain in the table. Callers expect the pending record when one exists.
	const q = `
		SELECT id, group_id, requester_user_id, message, status, created_at
		FROM join_requests
		WHERE group_id = $1 AND requester_user_id = $2
		ORDER BY CASE WHEN status = 'PENDING' THEN 0 ELSE 1 END, created_at DESC
		LIMIT 1
	`
	req, err := scanJoinRequest(infraPostgres.GetQuerier(ctx, r.db).QueryRow(ctx, q, groupID, userID.Value()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "JoinRequestRepository.FindByGroupAndUser failed", "error", err)
		return nil, apperror.NewInternal()
	}
	return req, nil
}

func (r *JoinRequestRepository) FindByGroup(ctx context.Context, groupID string, filters domainGroup.JoinRequestFilters) ([]*domainGroup.JoinRequest, int, error) {
	db := infraPostgres.GetQuerier(ctx, r.db)
	args := []any{groupID}
	idx := 2
	where := "group_id = $1"

	if filters.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", idx)
		args = append(args, filters.Status.String())
		idx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM join_requests WHERE %s", where)
	var total int
	// ErrNoRows only comes from a malformed group id, which has no requests: total stays zero.
	if err := db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		slog.ErrorContext(ctx, "JoinRequestRepository.FindByGroup count failed", "error", err)
		return nil, 0, apperror.NewInternal()
	}

	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.Limit < 1 {
		filters.Limit = 20
	}
	offset := (filters.Page - 1) * filters.Limit

	selectArgs := append(args, filters.Limit, offset)
	selectQuery := fmt.Sprintf(`
		SELECT id, group_id, requester_user_id, message, status, created_at
		FROM join_requests
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, idx, idx+1)

	rows, err := db.Query(ctx, selectQuery, selectArgs...)
	if err != nil {
		slog.ErrorContext(ctx, "JoinRequestRepository.FindByGroup query failed", "error", err)
		return nil, 0, apperror.NewInternal()
	}
	defer rows.Close()

	var out []*domainGroup.JoinRequest
	for rows.Next() {
		req, err := scanJoinRequest(rows)
		if err != nil {
			slog.ErrorContext(ctx, "JoinRequestRepository.FindByGroup scan failed", "error", err)
			return nil, 0, apperror.NewInternal()
		}
		out = append(out, req)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "JoinRequestRepository.FindByGroup rows error", "error", err)
		return nil, 0, apperror.NewInternal()
	}
	if out == nil {
		out = []*domainGroup.JoinRequest{}
	}
	return out, total, nil
}

func (r *JoinRequestRepository) Delete(ctx context.Context, id string) error {
	tag, err := infraPostgres.GetQuerier(ctx, r.db).Exec(ctx, `DELETE FROM join_requests WHERE id = $1 AND status = 'PENDING'`, id)
	if err != nil {
		slog.ErrorContext(ctx, "JoinRequestRepository.Delete failed", "error", err)
		return apperror.NewInternal()
	}
	if tag.RowsAffected() == 0 {
		return apperror.NewConflict(domainGroup.ErrCodeRequestAlreadyProcessed, "join request has already been processed")
	}
	return nil
}

type joinRequestScanner interface {
	Scan(dest ...any) error
}

func scanJoinRequest(row joinRequestScanner) (*domainGroup.JoinRequest, error) {
	var id, groupID, requesterUserID, status string
	var message *string
	var createdAt time.Time
	if err := row.Scan(&id, &groupID, &requesterUserID, &message, &status, &createdAt); err != nil {
		return nil, err
	}
	return domainGroup.RestoreJoinRequest(
		id,
		groupID,
		shared.RestoreUserID(requesterUserID),
		message,
		domainGroup.RestoreJoinRequestStatus(status),
		createdAt,
	), nil
}
