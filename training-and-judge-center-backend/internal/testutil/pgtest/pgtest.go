//go:build integration

package pgtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/goosemigrator"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	image  = "postgres:16-alpine"
	envURL = "TEST_DATABASE_URL"
)

var server pgtestdb.Config

func Main(m *testing.M) int {
	ctx := context.Background()

	dsn, stop, err := startServer(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pgtest:", err)
		return 1
	}
	defer stop()

	server, err = configFrom(dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pgtest:", err)
		return 1
	}

	return m.Run()
}

func NewDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if server.Host == "" {
		t.Fatal("pgtest: TestMain must return pgtest.Main(m)")
	}

	migrator := goosemigrator.New(".", goosemigrator.WithFS(os.DirFS(migrationsDir())))
	conf := pgtestdb.Custom(t, server, migrator)

	pool, err := pgxpool.New(context.Background(), conf.URL())
	if err != nil {
		t.Fatalf("pgtest: connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func startServer(ctx context.Context) (string, func(), error) {
	if url := os.Getenv(envURL); url != "" {
		return url, func() {}, nil
	}
	container, err := tcpostgres.Run(ctx, image,
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithCmd("postgres", "-c", "fsync=off"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(90*time.Second),
		),
	)
	if err != nil {
		return "", nil, fmt.Errorf("start postgres container: %w", err)
	}
	stop := func() { _ = container.Terminate(context.Background()) }
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		stop()
		return "", nil, fmt.Errorf("connection string: %w", err)
	}
	return dsn, stop, nil
}

func configFrom(dsn string) (pgtestdb.Config, error) {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return pgtestdb.Config{}, fmt.Errorf("parse dsn: %w", err)
	}
	return pgtestdb.Config{
		DriverName:                "pgx",
		Host:                      cfg.Host,
		Port:                      fmt.Sprint(cfg.Port),
		User:                      cfg.User,
		Password:                  cfg.Password,
		Database:                  cfg.Database,
		Options:                   "sslmode=disable",
		ForceTerminateConnections: true,
	}, nil
}

func migrationsDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "cmd", "migrate", "migrations")
}
