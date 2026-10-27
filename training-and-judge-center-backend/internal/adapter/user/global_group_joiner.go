package user

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// GlobalGroupJoiner implements application/user.GlobalGroupJoiner.
type GlobalGroupJoiner struct {
	db infraPostgres.Querier
}

func NewGlobalGroupJoiner(db infraPostgres.Querier) *GlobalGroupJoiner {
	return &GlobalGroupJoiner{db: db}
}

// AddToGlobalGroup inserts a MEMBER row for the default group, resolved by
// is_default rather than a hardcoded id. ON CONFLICT DO NOTHING makes it
// safe to call more than once for the same user (idempotent, mirrors the
// bootstrap migration's own guard).
func (j *GlobalGroupJoiner) AddToGlobalGroup(ctx context.Context, userID string) error {
	q := infraPostgres.GetQuerier(ctx, j.db)
	_, err := q.Exec(ctx, `
		INSERT INTO group_members (id, group_id, user_id, member_role, joined_at, join_method)
		SELECT $1, g.id, $2, 'MEMBER', NOW(), 'OPEN_JOIN'
		FROM groups g
		WHERE g.is_default = TRUE
		ON CONFLICT (group_id, user_id) DO NOTHING`,
		uuid.New().String(), userID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to add user to global group", "user_id", userID, "error", err)
		return apperror.NewInternal()
	}
	return nil
}
