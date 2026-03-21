package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/config"
	"github.com/training-judge-center/backend/internal/platform/email"
	jwtplatform "github.com/training-judge-center/backend/internal/platform/jwt"
	"github.com/training-judge-center/backend/internal/platform/postgres"
	"github.com/training-judge-center/backend/internal/platform/ratelimit"
	"github.com/training-judge-center/backend/internal/server"
	"github.com/redis/go-redis/v9"
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

	// Redis client
	redisOpt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		slog.Error("invalid redis URL", "error", err)
		os.Exit(1)
	}
	
	redisClient := redis.NewClient(redisOpt)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close() // Ahora se cerrará cuando el servidor (main) se detenga
	
	slog.Info("redis connected successfully")

	// Platform adapters
	userRepo := postgres.NewUserRepository(dbPool)
	_ = postgres.NewPasswordRecoveryRepository(dbPool) // Instantiated for future UseCases
	_ = postgres.NewEmailChangeRepository(dbPool)      // Instantiated for future UseCases
	jwtService := jwtplatform.NewService(cfg.JWTSecret, cfg.JWTExpirationHours)
	emailSender := email.NewSMTPSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom)
	_ = ratelimit.NewRedisRateLimiter(redisClient) // Instantiated for future UseCases

	// Use Cases
	createUserUC := appuser.NewCreateUserUseCase(userRepo)
	loginUC := appuser.NewLoginUseCase(userRepo, jwtService)
	getUserProfileUC := appuser.NewGetUserProfileUseCase(userRepo)
	updateUserUC := appuser.NewUpdateUserUseCase(userRepo)
	updatePasswordUC := appuser.NewUpdatePasswordUseCase(userRepo, emailSender)
	adminUpdateUserUC := appuser.NewAdminUpdateUserUseCase(userRepo)
	adminDeactivateUserUC := appuser.NewAdminDeactivateUserUseCase(userRepo)
	listUsersUC := appuser.NewListUsersUseCase(userRepo)

	// Handlers
	userHandler := handler.NewUserHandler(createUserUC, getUserProfileUC, updateUserUC, updatePasswordUC, adminUpdateUserUC, adminDeactivateUserUC, listUsersUC)
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

