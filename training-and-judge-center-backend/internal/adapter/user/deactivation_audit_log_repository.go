package user

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	domainUser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
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
		slog.ErrorContext(ctx, "database error saving deactivation audit log", "user_id", log.UserID(), "error", err)
		return apperror.NewInternal()
	}

	return nil
}
