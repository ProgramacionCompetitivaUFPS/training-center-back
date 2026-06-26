package material

import (
	"context"
	"fmt"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// DashboardProvider implements application/user.DashboardMaterialProvider.
type DashboardProvider struct {
	db infraPostgres.Querier
}

func NewDashboardProvider(db infraPostgres.Querier) *DashboardProvider {
	return &DashboardProvider{db: db}
}

func (p *DashboardProvider) GetRecentMaterials(ctx context.Context, userID string, limit int, windowDays int) ([]appuser.DashboardMaterial, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	rows, err := q.Query(ctx, fmt.Sprintf(`
		SELECT m.id,
		       m.title,
		       g.id   AS group_id,
		       g.name AS group_name,
		       m.published_at,
		       u.nickname AS author_nickname
		FROM materials m
		JOIN groups       g  ON g.id = m.group_id
		JOIN users        u  ON u.id = m.author_id
		JOIN group_members gm ON gm.group_id = m.group_id AND gm.user_id = $1
		WHERE m.status       = 'PUBLISHED'
		  AND m.published_at >= NOW() - INTERVAL '%d days'
		ORDER BY m.published_at DESC
		LIMIT $2`, windowDays),
		userID, limit,
	)
	if err != nil {
		slog.ErrorContext(ctx, "dashboard: failed to query recent materials", "user_id", userID, "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	var result []appuser.DashboardMaterial
	for rows.Next() {
		var m appuser.DashboardMaterial
		if err := rows.Scan(&m.ID, &m.Title, &m.GroupID, &m.GroupName, &m.PublishedAt, &m.AuthorNickname); err != nil {
			slog.ErrorContext(ctx, "dashboard: failed to scan material row", "error", err)
			return nil, apperror.NewInternal()
		}
		result = append(result, m)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "dashboard: materials rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}
