package user

import (
	"context"
	"fmt"

	domainUser "github.com/training-judge-center/backend/internal/domain/user"
	infraPostgres "github.com/training-judge-center/backend/internal/infrastructure/postgres"
)

type DeactivationAuditLogRepository struct {
	querier infraPostgres.Querier
}

func NewDeactivationAuditLogRepository(querier infraPostgres.Querier) *DeactivationAuditLogRepository {
	return &DeactivationAuditLogRepository{querier: querier}
}

func (r *DeactivationAuditLogRepository) Save(ctx context.Context, log *domainUser.DeactivationAuditLog) error {
	query := `
		INSERT INTO deactivation_audit_logs
		(id, user_id, original_email, original_nickname, occurred_at, ip, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	querier := infraPostgres.GetQuerier(ctx, r.querier)
	_, err := querier.Exec(ctx, query,
		log.ID(),
		log.UserID(),
		log.OriginalEmail(),
		log.OriginalNickname(),
		log.OccurredAt(),
		log.IP(),
		log.UserAgent(),
	)
	if err != nil {
		return fmt.Errorf("failed to save deactivation audit log: %w", err)
	}

	return nil
}
