package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/training-judge-center/backend/internal/domain/user"
)

type DeactivationAuditLogRepository struct {
	pool *pgxpool.Pool
}

func NewDeactivationAuditLogRepository(pool *pgxpool.Pool) *DeactivationAuditLogRepository {
	return &DeactivationAuditLogRepository{pool: pool}
}

func (r *DeactivationAuditLogRepository) Save(ctx context.Context, log *user.DeactivationAuditLog) error {
	query := `
		INSERT INTO deactivation_audit_logs 
		(id, user_id, original_email, original_nickname, occurred_at, ip, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.pool.Exec(ctx, query,
		log.ID,
		log.UserID,
		log.OriginalEmail,
		log.OriginalNickname,
		log.OccurredAt,
		log.IP,
		log.UserAgent,
	)
	if err != nil {
		return fmt.Errorf("failed to save deactivation audit log: %w", err)
	}

	return nil
}
