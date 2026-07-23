package contest

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appContest "github.com/training-judge-center/backend/internal/application/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ParticipantProfileProvider struct {
	db infraPostgres.Querier
}

func NewParticipantProfileProvider(db infraPostgres.Querier) *ParticipantProfileProvider {
	return &ParticipantProfileProvider{db: db}
}

func (p *ParticipantProfileProvider) GetProfiles(ctx context.Context, userIDs []string) (map[string]*appContest.ParticipantProfile, error) {
	profiles := make(map[string]*appContest.ParticipantProfile, len(userIDs))
	if len(userIDs) == 0 {
		return profiles, nil
	}

	q := infraPostgres.GetQuerier(ctx, p.db)
	rows, err := q.Query(ctx, `SELECT id, country, city, institution FROM users WHERE id = ANY($1)`, userIDs)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get participant profiles", "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	for rows.Next() {
		var profile appContest.ParticipantProfile
		if err := rows.Scan(&profile.ID, &profile.Country, &profile.City, &profile.Institution); err != nil {
			slog.ErrorContext(ctx, "failed to scan participant profile", "error", err)
			return nil, apperror.NewInternal()
		}
		profiles[profile.ID] = &profile
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "participant profiles rows error", "error", err)
		return nil, apperror.NewInternal()
	}

	return profiles, nil
}
