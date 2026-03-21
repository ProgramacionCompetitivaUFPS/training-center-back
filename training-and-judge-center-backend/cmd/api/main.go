package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/config"
	jwtplatform "github.com/training-judge-center/backend/internal/platform/jwt"
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

	// Platform adapters
	userRepo := postgres.NewUserRepository(dbPool)
	jwtService := jwtplatform.NewService(cfg.JWTSecret, cfg.JWTExpirationHours)

	// Use Cases
	createUserUC := appuser.NewCreateUserUseCase(userRepo)
	loginUC := appuser.NewLoginUseCase(userRepo, jwtService)
	getUserProfileUC := appuser.NewGetUserProfileUseCase(userRepo)
	updateUserUC := appuser.NewUpdateUserUseCase(userRepo)
	updatePasswordUC := appuser.NewUpdatePasswordUseCase(userRepo)
	adminUpdateUserUC := appuser.NewAdminUpdateUserUseCase(userRepo)
	adminDeactivateUserUC := appuser.NewAdminDeactivateUserUseCase(userRepo)

	// Handlers
	userHandler := handler.NewUserHandler(createUserUC, getUserProfileUC, updateUserUC, updatePasswordUC, adminUpdateUserUC, adminDeactivateUserUC)
	authHandler := handler.NewAuthHandler(loginUC)

	router := server.NewRouter(
		&server.Handlers{
			User: userHandler,
			Auth: authHandler,
		},
		&server.Services{
			TokenService: jwtService,
		},
	)

	slog.Info("server starting", "port", cfg.Port)

	if err := http.ListenAndServe(fmt.Sprintf(":%s", cfg.Port), router); err != nil {
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}

