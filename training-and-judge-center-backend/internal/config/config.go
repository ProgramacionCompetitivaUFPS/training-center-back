package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	DBHost             string
	DBPort             string
	DBUser             string
	DBPassword         string
	DBName             string
	MockAuth           bool
	StorageBackend     string
	StorageLocalDir    string
	GCSBucket          string
	VirtualObject      *VirtualObject
	JWTSecret          string
	JWTExpirationHours int
	SMTPHost           string
	SMTPPort           string
	SMTPUser           string
	SMTPPassword       string
	SMTPFrom           string
	RedisURL           string
	RabbitMQURL        string
	AllowedOrigins     []string
	FrontendBaseURL    string
}

func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		Port:               getEnv("PORT", "8080"),
		DBHost:             getEnv("DB_HOST", "localhost"),
		DBPort:             getEnv("DB_PORT", "5432"),
		DBUser:             getEnv("DB_USER", "postgres"),
		DBPassword:         getEnv("DB_PASSWORD", "postgres"),
		DBName:             getEnv("DB_NAME", "training_center"),
		MockAuth:           getEnv("MOCK_AUTH", "") == "1",
		StorageBackend:     getEnv("STORAGE_BACKEND", "local"),
		StorageLocalDir:    getEnv("STORAGE_LOCAL_DIR", ".local_storage"),
		GCSBucket:          getEnv("GCS_BUCKET", ""),
		VirtualObject:      loadVirtualObject(),
		JWTSecret:          getRequiredEnv("JWT_SECRET"),
		JWTExpirationHours: getEnvAsInt("JWT_EXPIRATION_HOURS", 24),
		SMTPHost:           getEnv("SMTP_HOST", "localhost"),
		SMTPPort:           getEnv("SMTP_PORT", "1025"),
		SMTPUser:           getEnv("SMTP_USER", ""),
		SMTPPassword:       getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:           getEnv("SMTP_FROM", "noreply@trainingcenter.com"),
		RedisURL:           getEnv("REDIS_URL", "localhost:6379"),
		RabbitMQURL:        getEnv("RABBITMQ_URL", ""),
		AllowedOrigins:     getEnvAsSlice("ALLOWED_ORIGINS", "http://localhost:5173"),
		FrontendBaseURL:    getEnv("FRONTEND_BASE_URL", "http://localhost:5173"),
	}
}

func getRequiredEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		slog.Error("required environment variable is not set — application cannot start", "key", key)
		os.Exit(1)
	}
	return value
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}

func getEnvAsSlice(key, fallback string) []string {
	value := os.Getenv(key)
	if value == "" {
		value = fallback
	}
	parts := strings.Split(value, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}
