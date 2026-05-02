package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/training-judge-center/backend/internal/config"
	infraPostgres "github.com/training-judge-center/backend/internal/infrastructure/postgres"
	domainUser "github.com/training-judge-center/backend/internal/domain/user"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	ctx := context.Background()

	email := requireEnv("ADMIN_EMAIL")
	password := requireEnv("ADMIN_PASSWORD")
	name := requireEnv("ADMIN_NAME")
	nickname := requireEnv("ADMIN_NICKNAME")
	country := getEnv("ADMIN_COUNTRY", "N/A")
	city := getEnv("ADMIN_CITY", "N/A")
	institution := getEnv("ADMIN_INSTITUTION", "N/A")

	if _, err := domainUser.NewEmail(email); err != nil {
		slog.Error("invalid ADMIN_EMAIL", "error", err)
		os.Exit(1)
	}
	if _, err := domainUser.NewPassword(password); err != nil {
		slog.Error("invalid ADMIN_PASSWORD", "error", err)
		os.Exit(1)
	}
	if _, err := domainUser.NewNickname(nickname); err != nil {
		slog.Error("invalid ADMIN_NICKNAME", "error", err)
		os.Exit(1)
	}

	passwordVO, _ := domainUser.NewPassword(password)

	pool, err := infraPostgres.NewConnectionPool(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	query := `
		INSERT INTO users (id, email, password, name, nickname, country, city, institution, role, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'ADMIN', 'ACTIVE', $9)
		ON CONFLICT (email) DO NOTHING`

	tag, err := pool.Exec(ctx, query,
		uuid.New().String(),
		email,
		passwordVO.Hash(),
		name,
		nickname,
		country,
		city,
		institution,
		time.Now(),
	)
	if err != nil {
		slog.Error("failed to insert admin user", "error", err)
		os.Exit(1)
	}

	if tag.RowsAffected() == 0 {
		slog.Info("admin user already exists, skipping", "email", email)
		return
	}

	slog.Info("admin user created", "email", email, "nickname", nickname)
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required env var is not set", "key", key)
		os.Exit(1)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
