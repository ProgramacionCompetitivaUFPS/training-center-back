package material

import (
	"context"
	"fmt"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// DashboardProvider implements application/user.DashboardMaterialProvider.
type DashboardProvider struct {
	db infraPostgres.Querier
}

func NewDashboardProvider(db infraPostgres.Querier) *DashboardProvider {
	return &DashboardProvider{db: db}
}

func (p *DashboardProvider) GetRecentMaterialsCount(ctx context.Context, userID string, windowDays int) (int, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	row := q.QueryRow(ctx, fmt.Sprintf(`
		SELECT COUNT(*)::INTEGER
		FROM materials m
		JOIN group_members gm ON gm.group_id = m.group_id AND gm.user_id = $1
		WHERE m.status       = 'PUBLISHED'
		  AND m.published_at >= NOW() - INTERVAL '%d days'`, windowDays),
		userID,
	)

	var count int
	if err := row.Scan(&count); err != nil {
		slog.ErrorContext(ctx, "dashboard: failed to query recent materials count", "user_id", userID, "error", err)
		return 0, apperror.NewInternal()
	}
	return count, nil
}
