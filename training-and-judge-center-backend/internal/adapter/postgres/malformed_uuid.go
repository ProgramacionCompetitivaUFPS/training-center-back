package postgres

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const InvalidTextRepresentation = "22P02" // PostgreSQL invalid_text_representation error code

// malformedUUIDAsNoRows makes reads by a value that isn't a valid uuid behave
// like reads by an id that doesn't exist. Postgres rejects such a value with
// invalid_text_representation, which every repository would otherwise turn
// into a 500, though a malformed id simply matches no row: QueryRow returns
// pgx.ErrNoRows and Query returns no rows.
//
// Exec is left alone: reporting "0 rows affected" for a failed write could
// make an INSERT look successful. Inside a transaction Postgres still aborts
// it, so a later statement in the same transaction fails as before.
type malformedUUIDAsNoRows struct {
	Querier
}

func (q malformedUUIDAsNoRows) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return malformedUUIDRow{Row: q.Querier.QueryRow(ctx, sql, args...), ctx: ctx}
}

func (q malformedUUIDAsNoRows) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	rows, err := q.Querier.Query(ctx, sql, args...)
	if err != nil && (rows == nil || !isMalformedUUID(err)) {
		return rows, err
	}
	if err != nil {
		logMalformedUUID(ctx, err)
	}
	return &malformedUUIDRows{Rows: rows, ctx: ctx}, nil
}

type malformedUUIDRow struct {
	pgx.Row
	ctx context.Context
}

func (r malformedUUIDRow) Scan(dest ...any) error {
	err := r.Row.Scan(dest...)
	if isMalformedUUID(err) {
		logMalformedUUID(r.ctx, err)
		return pgx.ErrNoRows
	}
	return err
}

type malformedUUIDRows struct {
	pgx.Rows
	ctx context.Context
}

func (r *malformedUUIDRows) Err() error {
	err := r.Rows.Err()
	if isMalformedUUID(err) {
		logMalformedUUID(r.ctx, err)
		return nil
	}
	return err
}

func isMalformedUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == InvalidTextRepresentation &&
		strings.Contains(pgErr.Message, "type uuid")
}

// logMalformedUUID keeps the translation visible, in case the bad value came
// from our own code rather than from a request.
func logMalformedUUID(ctx context.Context, err error) {
	slog.WarnContext(ctx, "postgres: malformed uuid treated as no rows", "error", err)
}
