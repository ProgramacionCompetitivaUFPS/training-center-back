package contest

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RegistrationRepository struct {
	db *pgxpool.Pool
}

func NewRegistrationRepository(db *pgxpool.Pool) *RegistrationRepository {
	return &RegistrationRepository{db: db}
}

func (r *RegistrationRepository) Save(ctx context.Context, reg *domainContest.ContestRegistration) error {
	q := infraPostgres.GetQuerier(ctx, r.db)
	_, err := q.Exec(ctx,
		`INSERT INTO contest_registrations (id, contest_id, user_id, registered_at)
		 VALUES ($1, $2, $3, $4)`,
		reg.ID(), reg.ContestID(), reg.UserID(), reg.RegisteredAt(),
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to save contest registration",
			"contest_id", reg.ContestID(), "user_id", reg.UserID(), "error", err)
		return apperror.NewInternal()
	}
	return nil
}

func (r *RegistrationRepository) ExistsByContestAndUser(ctx context.Context, contestID, userID string) (bool, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)
	var exists bool
	err := q.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM contest_registrations WHERE contest_id=$1 AND user_id=$2)`,
		contestID, userID,
	).Scan(&exists)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check contest registration existence",
			"contest_id", contestID, "user_id", userID, "error", err)
		return false, apperror.NewInternal()
	}
	return exists, nil
}

func (r *RegistrationRepository) CountByContest(ctx context.Context, contestID string) (int, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)
	var count int
	err := q.QueryRow(ctx,
		`SELECT COUNT(*) FROM contest_registrations WHERE contest_id=$1`,
		contestID,
	).Scan(&count)
	if err != nil {
		slog.ErrorContext(ctx, "failed to count contest registrations", "contest_id", contestID, "error", err)
		return 0, apperror.NewInternal()
	}
	return count, nil
}

func (r *RegistrationRepository) CountByContestBulk(ctx context.Context, contestIDs []string) (map[string]int, error) {
	result := make(map[string]int, len(contestIDs))
	for _, id := range contestIDs {
		result[id] = 0
	}
	if len(contestIDs) == 0 {
		return result, nil
	}
	q := infraPostgres.GetQuerier(ctx, r.db)
	rows, err := q.Query(ctx,
		`SELECT contest_id, COUNT(*) FROM contest_registrations WHERE contest_id = ANY($1) GROUP BY contest_id`,
		contestIDs,
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to bulk count contest registrations", "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()
	for rows.Next() {
		var contestID string
		var count int
		if err := rows.Scan(&contestID, &count); err != nil {
			slog.ErrorContext(ctx, "failed to scan bulk count row", "error", err)
			return nil, apperror.NewInternal()
		}
		result[contestID] = count
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "bulk count rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}

func (r *RegistrationRepository) ExistsByUserBulk(ctx context.Context, contestIDs []string, userID string) (map[string]bool, error) {
	result := make(map[string]bool, len(contestIDs))
	for _, id := range contestIDs {
		result[id] = false
	}
	if len(contestIDs) == 0 {
		return result, nil
	}
	q := infraPostgres.GetQuerier(ctx, r.db)
	rows, err := q.Query(ctx,
		`SELECT contest_id FROM contest_registrations WHERE contest_id = ANY($1) AND user_id = $2`,
		contestIDs, userID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to bulk check contest registrations", "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()
	for rows.Next() {
		var contestID string
		if err := rows.Scan(&contestID); err != nil {
			slog.ErrorContext(ctx, "failed to scan bulk exists row", "error", err)
			return nil, apperror.NewInternal()
		}
		result[contestID] = true
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "bulk exists rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}
