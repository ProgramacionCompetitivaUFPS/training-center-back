package group

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ContestRegistrationCleaner struct {
	db infraPostgres.Querier
}

func NewContestRegistrationCleaner(db infraPostgres.Querier) *ContestRegistrationCleaner {
	return &ContestRegistrationCleaner{db: db}
}

// DeleteScheduledByGroupAndUser removes all individual contest registrations for userID
// in SCHEDULED (not yet started) contests belonging to groupID.
func (c *ContestRegistrationCleaner) DeleteScheduledByGroupAndUser(ctx context.Context, groupID, userID string) (int, error) {
	tag, err := infraPostgres.GetQuerier(ctx, c.db).Exec(ctx, `
		DELETE FROM contest_registrations cr
		USING contests co
		WHERE cr.contest_id = co.id
		  AND co.group_id   = $1
		  AND cr.user_id    = $2
		  AND co.start_time > NOW()
	`, groupID, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to clean contest registrations on member removal",
			"group_id", groupID, "user_id", userID, "error", err)
		return 0, apperror.NewInternal()
	}
	return int(tag.RowsAffected()), nil
}
