package contest

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appContest "github.com/training-judge-center/backend/internal/application/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type TeamDisplayProvider struct {
	db infraPostgres.Querier
}

func NewTeamDisplayProvider(db infraPostgres.Querier) *TeamDisplayProvider {
	return &TeamDisplayProvider{db: db}
}

func (p *TeamDisplayProvider) GetDisplays(ctx context.Context, teamIDs []string) (map[string]*appContest.TeamDisplay, error) {
	displays := make(map[string]*appContest.TeamDisplay, len(teamIDs))
	if len(teamIDs) == 0 {
		return displays, nil
	}

	q := infraPostgres.GetQuerier(ctx, p.db)
	rows, err := q.Query(ctx, `SELECT id, name FROM teams WHERE id = ANY($1)`, teamIDs)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get team displays", "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	for rows.Next() {
		var display appContest.TeamDisplay
		if err := rows.Scan(&display.ID, &display.Name); err != nil {
			slog.ErrorContext(ctx, "failed to scan team display", "error", err)
			return nil, apperror.NewInternal()
		}
		displays[display.ID] = &display
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "team displays rows error", "error", err)
		return nil, apperror.NewInternal()
	}

	return displays, nil
}
