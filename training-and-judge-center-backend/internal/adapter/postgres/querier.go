package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row
}

// GetQuerier returns the transaction in ctx, or defaultQuerier. Outside a
// transaction, reads by a malformed uuid come back as "no rows" (see
// malformedUUIDAsNoRows); inside one the error is kept, because Postgres has
// already aborted the transaction and "no rows" would hide that.
func GetQuerier(ctx context.Context, defaultQuerier Querier) Querier {
	if tx := TxFromContext(ctx); tx != nil {
		return tx
	}
	return malformedUUIDAsNoRows{defaultQuerier}
}
