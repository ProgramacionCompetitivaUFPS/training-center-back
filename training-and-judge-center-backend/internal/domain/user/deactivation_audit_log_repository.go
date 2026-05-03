package user

import "context"

type DeactivationAuditLogRepository interface {
	Save(ctx context.Context, log *DeactivationAuditLog) error
}
