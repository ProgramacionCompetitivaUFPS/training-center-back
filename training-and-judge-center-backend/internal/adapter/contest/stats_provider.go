package contest

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// StatsProvider implements application/user.ContestParticipationProvider.
type StatsProvider struct {
	db infraPostgres.Querier
}

func NewStatsProvider(db infraPostgres.Querier) *StatsProvider {
	return &StatsProvider{db: db}
}

// GetContestsParticipatedCount counts distinct contests the user is or was
// registered to, individually or as part of a team, regardless of status.
func (p *StatsProvider) GetContestsParticipatedCount(ctx context.Context, userID string) (int, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	row := q.QueryRow(ctx, `
		SELECT COUNT(*)::INTEGER
		FROM contests c
		WHERE `+participationFilter,
		userID,
	)

	var count int
	if err := row.Scan(&count); err != nil {
		slog.ErrorContext(ctx, "profile stats: failed to query contests participated count", "user_id", userID, "error", err)
		return 0, apperror.NewInternal()
	}
	return count, nil
}
