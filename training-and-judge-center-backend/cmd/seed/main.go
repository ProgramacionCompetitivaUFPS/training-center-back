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
	ctx := context.Background()

	slog.Info("seed: starting admin user bootstrap")

	email := requireEnv("ADMIN_EMAIL")
	password := requireEnv("ADMIN_PASSWORD")
	name := getEnv("ADMIN_NAME", "admin")
	nickname := getEnv("ADMIN_NICKNAME", "admin")
	country := getEnv("ADMIN_COUNTRY", "admin")
	city := getEnv("ADMIN_CITY", "admin")
	institution := getEnv("ADMIN_INSTITUTION", "admin")

	if _, err := domainUser.NewEmail(email); err != nil {
		slog.Error("invalid ADMIN_EMAIL", "error", err)
		os.Exit(1)
	}
	passwordVO, err := domainUser.NewPassword(password)
	if err != nil {
		slog.Error("invalid ADMIN_PASSWORD", "error", err)
		os.Exit(1)
	}
	if _, err := domainUser.NewNickname(nickname); err != nil {
		slog.Error("invalid ADMIN_NICKNAME", "error", err)
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

	tag, err := pool.Exec(ctx,
		`UPDATE users SET password = $1 WHERE email = $2`,
		passwordVO.Hash(), email,
	)
	if err != nil {
		slog.Error("failed to update admin user", "error", err)
		os.Exit(1)
	}
	if tag.RowsAffected() > 0 {
		slog.Info("admin password updated", "email", email)
		return
	}

	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, email, password, name, nickname, country, city, institution, role, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'ADMIN', 'ACTIVE', $9)`,
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
