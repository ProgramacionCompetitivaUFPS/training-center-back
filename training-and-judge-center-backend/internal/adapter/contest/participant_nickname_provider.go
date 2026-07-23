package contest

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ParticipantNicknameProvider struct {
	db infraPostgres.Querier
}

func NewParticipantNicknameProvider(db infraPostgres.Querier) *ParticipantNicknameProvider {
	return &ParticipantNicknameProvider{db: db}
}

func (p *ParticipantNicknameProvider) GetNicknamesByIDs(ctx context.Context, userIDs []string) (map[string]string, error) {
	result := make(map[string]string, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	q := infraPostgres.GetQuerier(ctx, p.db)
	rows, err := q.Query(ctx, `SELECT id, nickname FROM users WHERE id = ANY($1)`, userIDs)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch participant nicknames", "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()
	for rows.Next() {
		var id, nickname string
		if err := rows.Scan(&id, &nickname); err != nil {
			slog.ErrorContext(ctx, "failed to scan nickname row", "error", err)
			return nil, apperror.NewInternal()
		}
		result[id] = nickname
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "nickname rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}
