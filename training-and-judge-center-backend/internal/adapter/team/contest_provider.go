package team

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ContestProvider struct {
	db infraPostgres.Querier
}

func NewContestProvider(db infraPostgres.Querier) *ContestProvider {
	return &ContestProvider{db: db}
}

func (p *ContestProvider) GetContestInfo(ctx context.Context, contestID string) (*appTeam.ContestInfo, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	var groupID, mode string
	var groupIsVisible bool
	var sizeMin, sizeMax int
	var startTime, endTime time.Time

	err := q.QueryRow(ctx,
		`SELECT c.group_id, g.visibility = 'VISIBLE', c.participation_mode, c.team_size_min, c.team_size_max, c.start_time, c.end_time
		 FROM contests c JOIN groups g ON g.id = c.group_id WHERE c.id=$1`,
		contestID,
	).Scan(&groupID, &groupIsVisible, &mode, &sizeMin, &sizeMax, &startTime, &endTime)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
		}
		slog.ErrorContext(ctx, "failed to get contest info for team", "contest_id", contestID, "error", err)
		return nil, apperror.NewInternal()
	}

	now := time.Now().UTC()
	status := "SCHEDULED"
	if !now.Before(startTime) {
		if now.After(endTime) {
			status = "FINISHED"
		} else {
			status = "ACTIVE"
		}
	}

	return &appTeam.ContestInfo{
		ID:                contestID,
		GroupID:           groupID,
		GroupIsVisible:    groupIsVisible,
		ParticipationMode: mode,
		TeamSizeMin:       sizeMin,
		TeamSizeMax:       sizeMax,
		Status:            status,
	}, nil
}
