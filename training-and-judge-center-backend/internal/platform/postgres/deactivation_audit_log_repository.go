package postgres

import (
	"context"
	"fmt"

	"github.com/training-judge-center/backend/internal/domain/user"
)

type DeactivationAuditLogRepository struct {
	querier Querier
}

func NewDeactivationAuditLogRepository(querier Querier) *DeactivationAuditLogRepository {
	return &DeactivationAuditLogRepository{querier: querier}
}

func (r *DeactivationAuditLogRepository) Save(ctx context.Context, log *user.DeactivationAuditLog) error {
	query := `
		INSERT INTO deactivation_audit_logs
		(id, user_id, original_email, original_nickname, occurred_at, ip, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.querier.Exec(ctx, query,
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
