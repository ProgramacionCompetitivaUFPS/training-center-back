package main

import (
	"context"
	"embed"
	"log/slog"
	"os"

	"github.com/pressly/goose/v3"
	"github.com/training-judge-center/backend/internal/config"
	"github.com/training-judge-center/backend/internal/platform/postgres"
)

//go:embed migrations/*.sql
var migrations embed.FS

func main() {
	if len(os.Args) < 2 {
		slog.Error("usage: go run cmd/migrate/main.go [up|down|status]")
		os.Exit(1)
	}

	command := os.Args[1]
	cfg := config.Load()
	ctx := context.Background()

	pool, err := postgres.NewConnectionPool(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	gooseDB, err := goose.OpenDBWithDriver("postgres", pool.Config().ConnString())
	if err != nil {
		slog.Error("failed to open db for goose", "error", err)
		os.Exit(1)
	}
	defer gooseDB.Close()

	goose.SetBaseFS(migrations)

	if err := goose.SetDialect("postgres"); err != nil {
		slog.Error("failed to set dialect", "error", err)
		os.Exit(1)
	}

	if err := goose.RunContext(ctx, command, gooseDB, "migrations"); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
}
