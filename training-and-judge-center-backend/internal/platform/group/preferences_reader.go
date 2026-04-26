package group

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/training-judge-center/backend/pkg/apperror"
)

type PreferencesReader struct {
	db *pgxpool.Pool
}

func NewPreferencesReader(db *pgxpool.Pool) *PreferencesReader {
	return &PreferencesReader{db: db}
}

type userPreferences struct {
	HideGlobalGroup bool `json:"hideGlobalGroup"`
}

func (p *PreferencesReader) HideGlobalGroup(ctx context.Context, userID string) (bool, error) {
	var raw []byte
	err := p.db.QueryRow(ctx, `SELECT preferences FROM users WHERE id = $1`, userID).Scan(&raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		slog.ErrorContext(ctx, "HideGlobalGroup failed", "error", err, "user_id", userID)
		return false, apperror.NewInternal()
	}
	if len(raw) == 0 {
		return false, nil
	}
	var prefs userPreferences
	if err := json.Unmarshal(raw, &prefs); err != nil {
		slog.WarnContext(ctx, "user preferences JSON is invalid; defaulting hideGlobalGroup=false", "user_id", userID, "error", err)
		return false, nil
	}
	return prefs.HideGlobalGroup, nil
}
