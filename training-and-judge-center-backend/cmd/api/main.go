package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/config"
	"github.com/training-judge-center/backend/internal/domain/user"
	platformConfig "github.com/training-judge-center/backend/internal/platform/config"
	"github.com/training-judge-center/backend/internal/platform/postgres"
	platformUser "github.com/training-judge-center/backend/internal/platform/user"
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


	problemRepo := postgres.NewProblemRepository(dbPool)
	settingsProvider := platformConfig.NewVirtualObjectProvider(cfg.VirtualObject)
	
	var displayProvider user.DisplayProvider = nil


	createProblemUseCase := appProblem.NewCreateProblemUseCase(problemRepo, settingsProvider)

	if cfg.MockAuth {
		slog.Info("running in MOCK_AUTH mode")
		displayProvider = platformUser.NewMockDisplayProvider()
		
		problemHandler := handler.NewProblemHandler(createProblemUseCase, displayProvider)
		router := server.NewRouter(cfg, &server.Handlers{Problem: problemHandler})

		slog.Info("server starting", "port", cfg.Port)
		if err := http.ListenAndServe(fmt.Sprintf(":%s", cfg.Port), router); err != nil {
			slog.Error("server failed to start", "error", err)
			os.Exit(1)
		}
		return
	}

	problemHandler := handler.NewProblemHandler(createProblemUseCase, displayProvider)

	router := server.NewRouter(cfg, &server.Handlers{
		Problem: problemHandler,
	})

	slog.Info("server starting", "port", cfg.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", cfg.Port), router); err != nil {
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}
