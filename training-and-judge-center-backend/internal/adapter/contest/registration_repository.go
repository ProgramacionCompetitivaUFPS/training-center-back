package contest

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RegistrationRepository struct {
	db infraPostgres.Querier
}

func NewRegistrationRepository(db infraPostgres.Querier) *RegistrationRepository {
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
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil
		}
		slog.ErrorContext(ctx, "failed to save contest registration",
			"contest_id", reg.ContestID(), "user_id", reg.UserID(), "error", err)
		return apperror.NewInternal()
	}
	return nil
}

func (r *RegistrationRepository) FindByContestAndUser(ctx context.Context, contestID, userID string) (*domainContest.ContestRegistration, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)
	var id, cID, uID string
	var registeredAt time.Time
	err := q.QueryRow(ctx,
		`SELECT id, contest_id, user_id, registered_at FROM contest_registrations WHERE contest_id=$1 AND user_id=$2`,
		contestID, userID,
	).Scan(&id, &cID, &uID, &registeredAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "failed to find contest registration",
			"contest_id", contestID, "user_id", userID, "error", err)
		return nil, apperror.NewInternal()
	}
	return domainContest.RestoreContestRegistration(id, cID, uID, registeredAt), nil
}

func (r *RegistrationRepository) Delete(ctx context.Context, contestID, userID string) error {
	q := infraPostgres.GetQuerier(ctx, r.db)
	tag, err := q.Exec(ctx,
		`DELETE FROM contest_registrations WHERE contest_id=$1 AND user_id=$2`,
		contestID, userID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to delete contest registration",
			"contest_id", contestID, "user_id", userID, "error", err)
		return apperror.NewInternal()
	}
	if tag.RowsAffected() == 0 {
		return apperror.NewNotFound(domainContest.ErrCodeNotRegistered, "you are not registered to this contest")
	}
	return nil
}

func (r *RegistrationRepository) ListByContest(ctx context.Context, contestID string, page, limit int) ([]*domainContest.ContestRegistration, int, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)
	var total int
	err := q.QueryRow(ctx,
		`SELECT COUNT(*) FROM contest_registrations WHERE contest_id=$1`,
		contestID,
	).Scan(&total)
	if err != nil {
		slog.ErrorContext(ctx, "failed to count contest registrations for list", "contest_id", contestID, "error", err)
		return nil, 0, apperror.NewInternal()
	}

	offset := (page - 1) * limit
	rows, err := q.Query(ctx,
		`SELECT id, contest_id, user_id, registered_at FROM contest_registrations WHERE contest_id=$1 ORDER BY registered_at ASC LIMIT $2 OFFSET $3`,
		contestID, limit, offset,
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list contest registrations", "contest_id", contestID, "error", err)
		return nil, 0, apperror.NewInternal()
	}
	defer rows.Close()

	var regs []*domainContest.ContestRegistration
	for rows.Next() {
		var id, cID, uID string
		var registeredAt time.Time
		if err := rows.Scan(&id, &cID, &uID, &registeredAt); err != nil {
			slog.ErrorContext(ctx, "failed to scan registration row", "contest_id", contestID, "error", err)
			return nil, 0, apperror.NewInternal()
		}
		regs = append(regs, domainContest.RestoreContestRegistration(id, cID, uID, registeredAt))
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "list registrations rows error", "contest_id", contestID, "error", err)
		return nil, 0, apperror.NewInternal()
	}
	return regs, total, nil
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
		return result, apperror.NewInternal()
	}
	defer rows.Close()
	for rows.Next() {
		var contestID string
		var count int
		if err := rows.Scan(&contestID, &count); err != nil {
			slog.ErrorContext(ctx, "failed to scan bulk count row", "error", err)
			return result, apperror.NewInternal()
		}
		result[contestID] = count
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "bulk count rows error", "error", err)
		return result, apperror.NewInternal()
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
		return result, apperror.NewInternal()
	}
	defer rows.Close()
	for rows.Next() {
		var contestID string
		if err := rows.Scan(&contestID); err != nil {
			slog.ErrorContext(ctx, "failed to scan bulk exists row", "error", err)
			return result, apperror.NewInternal()
		}
		result[contestID] = true
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "bulk exists rows error", "error", err)
		return result, apperror.NewInternal()
	}
	return result, nil
}
