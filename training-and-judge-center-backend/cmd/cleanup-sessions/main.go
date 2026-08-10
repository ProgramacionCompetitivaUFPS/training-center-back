package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	adapterUser "github.com/training-judge-center/backend/internal/adapter/user"
	"github.com/training-judge-center/backend/internal/config"
)

func main() {
	_ = godotenv.Load()
	ctx := context.Background()

	retentionDays := getEnvAsInt("CLEANUP_RETENTION_DAYS", 15)

	cfg := &config.Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "training_center"),
	}

	pool, err := infraPostgres.NewConnectionPool(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)
	repo := adapterUser.NewRefreshTokenRepository(pool)

	if err := repo.DeleteRevokedOrExpiredBefore(ctx, cutoff); err != nil {
		slog.Error("cleanup-sessions: failed to delete old refresh tokens", "error", err)
		os.Exit(1)
	}

	slog.Info("cleanup-sessions: done", "cutoff", cutoff, "retention_days", retentionDays)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if intVal, err := strconv.Atoi(v); err == nil {
			return intVal
		}
	}
	return fallback
}
