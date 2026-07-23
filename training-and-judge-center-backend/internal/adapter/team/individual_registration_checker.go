package team

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type IndividualRegistrationChecker struct {
	db infraPostgres.Querier
}

func NewIndividualRegistrationChecker(db infraPostgres.Querier) *IndividualRegistrationChecker {
	return &IndividualRegistrationChecker{db: db}
}

func (c *IndividualRegistrationChecker) AreUsersRegisteredIndividually(ctx context.Context, contestID string, userIDs []string) (map[string]bool, error) {
	result := make(map[string]bool, len(userIDs))
	for _, id := range userIDs {
		result[id] = false
	}
	if len(userIDs) == 0 {
		return result, nil
	}

	q := infraPostgres.GetQuerier(ctx, c.db)
	rows, err := q.Query(ctx,
		`SELECT user_id FROM contest_registrations WHERE contest_id=$1 AND user_id = ANY($2::UUID[])`,
		contestID, userIDs,
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check individual registrations",
			"contest_id", contestID, "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			slog.ErrorContext(ctx, "failed to scan individual registration row", "error", err)
			return nil, apperror.NewInternal()
		}
		result[userID] = true
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "individual registrations rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}
