//go:build integration

package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/training-judge-center/backend/internal/config"
	"github.com/training-judge-center/backend/internal/testutil/pgtest"
)

func TestMain(m *testing.M) {
	os.Exit(pgtest.Main(m))
}

const itUser = "11111111-1111-1111-1111-111111111111"

func countUsers(t *testing.T, ctx context.Context, q Querier) int {
	t.Helper()
	var n int
	require.NoError(t, q.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE id = $1`, itUser).Scan(&n))
	return n
}

func insertUser(t *testing.T, ctx context.Context, q Querier) {
	t.Helper()
	_, err := q.Exec(ctx, `
		INSERT INTO users (id, email, password, name, nickname, country, city, institution)
		VALUES ($1, 'ana@example.com', 'hash', 'ana', 'ana', 'Colombia', 'Bogota', 'UNAL')`, itUser)
	require.NoError(t, err)
}

func TestTransactionManager_WithTx_CommitsWhenTheFunctionSucceeds(t *testing.T) {
	pool := pgtest.NewDB(t)
	tm := NewTransactionManager(pool)

	err := tm.WithTx(context.Background(), func(txCtx context.Context) error {
		insertUser(t, txCtx, GetQuerier(txCtx, pool))
		assert.Equal(t, 0, countUsers(t, context.Background(), pool), "not visible outside until it commits")
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, countUsers(t, context.Background(), pool))
}

func TestTransactionManager_WithTx_RollsBackAndReturnsTheErrorWhenTheFunctionFails(t *testing.T) {
	pool := pgtest.NewDB(t)
	tm := NewTransactionManager(pool)
	boom := errors.New("boom")

	err := tm.WithTx(context.Background(), func(txCtx context.Context) error {
		insertUser(t, txCtx, GetQuerier(txCtx, pool))
		return boom
	})

	require.ErrorIs(t, err, boom)
	assert.Equal(t, 0, countUsers(t, context.Background(), pool))
}

func TestTransactionManager_WithTx_RollsBackWhenTheFunctionPanics(t *testing.T) {
	pool := pgtest.NewDB(t)
	tm := NewTransactionManager(pool)

	assert.Panics(t, func() {
		_ = tm.WithTx(context.Background(), func(txCtx context.Context) error {
			insertUser(t, txCtx, GetQuerier(txCtx, pool))
			panic("handler blew up")
		})
	})

	assert.Equal(t, 0, countUsers(t, context.Background(), pool))
}

func TestTransactionManager_WithTx_FailsWhenTheDatabaseIsClosed(t *testing.T) {
	pool := pgtest.NewDB(t)
	tm := NewTransactionManager(pool)
	pool.Close()

	err := tm.WithTx(context.Background(), func(context.Context) error {
		t.Fatal("the function must not run when the transaction could not begin")
		return nil
	})

	require.Error(t, err)
}

func TestGetQuerier_UsesTheTransactionInsideAndTheMalformedUUIDWrapperOutside(t *testing.T) {
	pool := pgtest.NewDB(t)
	tm := NewTransactionManager(pool)

	var outside Querier = GetQuerier(context.Background(), pool)
	err := outside.QueryRow(context.Background(), `SELECT 1 FROM users WHERE id = $1`, "not-a-uuid").Scan(new(int))
	assert.ErrorIs(t, err, pgx.ErrNoRows, "outside a transaction a malformed uuid reads as no rows")

	var inTx error
	_ = tm.WithTx(context.Background(), func(txCtx context.Context) error {
		assert.NotNil(t, TxFromContext(txCtx))
		inside := GetQuerier(txCtx, pool)
		inTx = inside.QueryRow(txCtx, `SELECT 1 FROM users WHERE id = $1`, "not-a-uuid").Scan(new(int))
		return inTx
	})
	require.Error(t, inTx)
	assert.NotErrorIs(t, inTx, pgx.ErrNoRows, "inside a transaction the error is kept: it is already aborted")
	assert.Nil(t, TxFromContext(context.Background()))
}

func TestNewConnectionPool_UnreachableDatabaseReturnsAnError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := NewConnectionPool(ctx, &config.Config{DBHost: "127.0.0.1", DBPort: "1", DBUser: "u", DBPassword: "p", DBName: "d"})

	require.Error(t, err)
	assert.Nil(t, pool)
	assert.Contains(t, err.Error(), "unable to ping database")
}
