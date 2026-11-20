package contest

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5/pgconn"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const lockNotAvailable = "55P03"

type RegistrationLocker struct {
	db infraPostgres.Querier
}

func NewRegistrationLocker(db infraPostgres.Querier) *RegistrationLocker {
	return &RegistrationLocker{db: db}
}

// Lock takes a transaction-scoped advisory lock for the contest and gives up after 3s,
// so a burst of registrations never keeps a pooled connection waiting indefinitely.
func (l *RegistrationLocker) Lock(ctx context.Context, contestID string) error {
	q := infraPostgres.GetQuerier(ctx, l.db)
	if _, err := q.Exec(ctx, `SET LOCAL lock_timeout = '3s'`); err != nil {
		slog.ErrorContext(ctx, "failed to set the registration lock timeout", "contest_id", contestID, "error", err)
		return apperror.NewInternal()
	}
	if _, err := q.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, contestID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == lockNotAvailable {
			return apperror.NewServiceUnavailable(domainContest.ErrCodeRegistrationBusy,
				"too many registrations at the same time; please try again")
		}
		slog.ErrorContext(ctx, "failed to take the registration lock", "contest_id", contestID, "error", err)
		return apperror.NewInternal()
	}
	return nil
}
