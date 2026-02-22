package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/config"
	"github.com/training-judge-center/backend/internal/platform/postgres"
	"github.com/training-judge-center/backend/internal/server"
	"github.com/training-judge-center/backend/internal/server/handler"
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

	slog.Info("database connected successfully")

	userRepo := postgres.NewUserRepository(dbPool)
	createUserUC := appuser.NewCreateUserUseCase(userRepo)
	userHandler := handler.NewUserHandler(createUserUC)

	router := server.NewRouter(&server.Handlers{
		User: userHandler,
	})

	slog.Info("server starting", "port", cfg.Port)

	if err := http.ListenAndServe(fmt.Sprintf(":%s", cfg.Port), router); err != nil {
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}
