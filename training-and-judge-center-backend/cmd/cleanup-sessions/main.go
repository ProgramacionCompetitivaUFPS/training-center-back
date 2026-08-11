package main

import (
	"context"
	"fmt"
	"log/slog"
	"math"
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

	retentionDays, err := getEnvAsPositiveInt("CLEANUP_RETENTION_DAYS", 15)
	if err != nil {
		slog.Error("cleanup-sessions: invalid CLEANUP_RETENTION_DAYS", "error", err)
		os.Exit(1)
	}

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

func getEnvAsPositiveInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	intVal, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	if intVal < 1 {
		return 0, fmt.Errorf("%s must be >= 1, got %d", key, intVal)
	}
	if intVal > math.MaxInt/24 {
		return 0, fmt.Errorf("%s is too large: %d", key, intVal)
	}
	return intVal, nil
}
