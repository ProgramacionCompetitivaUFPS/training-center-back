package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	errMalformedUUID = &pgconn.PgError{Code: InvalidTextRepresentation, Message: `invalid input syntax for type uuid: "96b"`}
	errMalformedInt  = &pgconn.PgError{Code: InvalidTextRepresentation, Message: `invalid input syntax for type integer: "x"`}
	errOther         = errors.New("connection reset")
)

type fakeRow struct{ err error }

func (r fakeRow) Scan(...any) error { return r.err }

type fakeRows struct {
	pgx.Rows
	err error
}

func (r *fakeRows) Next() bool { return false }
func (r *fakeRows) Err() error { return r.err }
func (r *fakeRows) Close()     {}

type fakeQuerier struct {
	rowErr   error
	rowsErr  error
	queryErr error
	execErr  error
}

func (q fakeQuerier) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, q.execErr
}
func (q fakeQuerier) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return &fakeRows{err: q.rowsErr}, q.queryErr
}
func (q fakeQuerier) QueryRow(context.Context, string, ...interface{}) pgx.Row {
	return fakeRow{err: q.rowErr}
}

func TestMalformedUUIDAsNoRows_QueryRow(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{"malformed uuid reads as no rows", errMalformedUUID, pgx.ErrNoRows},
		{"other invalid input is kept", errMalformedInt, errMalformedInt},
		{"other errors are kept", errOther, errOther},
		{"success is kept", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := GetQuerier(context.Background(), fakeQuerier{rowErr: tt.err})
			if got := q.QueryRow(context.Background(), "").Scan(); !errors.Is(got, tt.want) {
				t.Errorf("Scan() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMalformedUUIDAsNoRows_Query(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{"malformed uuid reads as no rows", errMalformedUUID, nil},
		{"other invalid input is kept", errMalformedInt, errMalformedInt},
		{"other errors are kept", errOther, errOther},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := GetQuerier(context.Background(), fakeQuerier{rowsErr: tt.err})
			rows, err := q.Query(context.Background(), "")
			if err != nil {
				t.Fatalf("Query() error = %v", err)
			}
			if rows.Next() {
				t.Error("expected no rows")
			}
			if got := rows.Err(); !errors.Is(got, tt.want) {
				t.Errorf("Err() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMalformedUUIDAsNoRows_QueryFailingUpfront(t *testing.T) {
	q := GetQuerier(context.Background(), fakeQuerier{queryErr: errOther, rowsErr: errOther})
	if _, err := q.Query(context.Background(), ""); !errors.Is(err, errOther) {
		t.Errorf("Query() error = %v, want %v", err, errOther)
	}
}

func TestMalformedUUIDAsNoRows_ExecKeepsTheError(t *testing.T) {
	q := GetQuerier(context.Background(), fakeQuerier{execErr: errMalformedUUID})
	if _, err := q.Exec(context.Background(), ""); !errors.Is(err, errMalformedUUID) {
		t.Errorf("Exec() error = %v, want the original error", err)
	}
}

type fakeTx struct {
	pgx.Tx
	rowErr error
}

func (t fakeTx) QueryRow(context.Context, string, ...interface{}) pgx.Row {
	return fakeRow{err: t.rowErr}
}

func TestGetQuerier_KeepsTheErrorInsideATransaction(t *testing.T) {
	ctx := context.WithValue(context.Background(), txContextKey, pgx.Tx(fakeTx{rowErr: errMalformedUUID}))
	q := GetQuerier(ctx, fakeQuerier{})
	if got := q.QueryRow(ctx, "").Scan(); !errors.Is(got, errMalformedUUID) {
		t.Errorf("Scan() = %v, want the original error: the transaction is already aborted", got)
	}
}
