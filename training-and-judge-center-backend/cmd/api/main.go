package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/training-judge-center/backend/internal/config"
	"github.com/training-judge-center/backend/internal/platform/postgres"
	"github.com/training-judge-center/backend/internal/server"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	dbPool, err := postgres.NewConnectionPool(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()
	a := 3
	b := 4
	slog.Info("database connected successfully")

	router := server.NewRouter()

	slog.Info("server starting", "port", cfg.Port)
	slog.Info(a + b)

	if err := http.ListenAndServe(fmt.Sprintf(":%s", cfg.Port), router); err != nil {
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}
