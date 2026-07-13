package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	googleStorage "cloud.google.com/go/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/moby/moby/client"
	"gopkg.in/yaml.v3"

	adapterjudge "github.com/training-judge-center/backend/internal/adapter/judge"
	judgepool "github.com/training-judge-center/backend/internal/adapter/judge/pool"
	infrapostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	adapterqueue "github.com/training-judge-center/backend/internal/adapter/queue"
	adaptersubmission "github.com/training-judge-center/backend/internal/adapter/submission"
	appjudge "github.com/training-judge-center/backend/internal/application/judge"
	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
)

func main() {
	ctx := context.Background()

	// infrastructure

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", "postgres"),
		getEnv("DB_NAME", "training_center"),
	)
	dbPool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		slog.Error("worker: failed to create db pool", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()
	if err := dbPool.Ping(ctx); err != nil {
		slog.Error("worker: failed to ping db", "error", err)
		os.Exit(1)
	}
	slog.Info("worker: database connected")

	txManager := infrapostgres.NewTransactionManager(dbPool)

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		slog.Error("worker: failed to create docker client", "error", err)
		os.Exit(1)
	}
	defer dockerClient.Close()

	rabbitmqURL := getRequiredEnv("RABBITMQ_URL")
	queue, err := adapterqueue.NewRabbitMQQueue(rabbitmqURL)
	if err != nil {
		slog.Error("worker: failed to connect to rabbitmq", "error", err)
		os.Exit(1)
	}
	defer queue.Close()
	slog.Info("worker: rabbitmq connected")

	storageBackend := getEnv("STORAGE_BACKEND", "local")
	storageLocalDir := getEnv("STORAGE_LOCAL_DIR", ".local_storage")
	gcsBucket := getEnv("GCS_BUCKET", "")

	// judge adapters

	judgeCfg := loadJudgeConfig()

	poolMemLimit := getRequiredEnvInt64("POD_MEMORY_LIMIT")

	poolLanguages := make(map[string]judgepool.LanguageConfig, len(judgeCfg.Judge.Languages))
	execLanguages := make(map[string]adapterjudge.LanguageExecConfig, len(judgeCfg.Judge.Languages))
	for lang, lc := range judgeCfg.Judge.Languages {
		poolLanguages[lang] = judgepool.LanguageConfig{
			Image:       lc.Image,
			CPU:         lc.CPU,
			MemoryBytes: lc.MemoryBytes,
		}
		execLanguages[lang] = adapterjudge.LanguageExecConfig{
			CompileCmd: lc.CompileCmd,
			RunCmd:     lc.RunCmd,
			Extension:  lc.Extension,
		}
	}

	poolCfg := judgepool.PoolConfig{
		MemLimitBytes: poolMemLimit,
		OverheadBytes: judgeCfg.Judge.MemoryOverheadBytes,
		IdleTimeout:   time.Duration(judgeCfg.Judge.IdleTimeoutMinutes) * time.Minute,
		ReapInterval:  time.Minute,
		Languages:     poolLanguages,
	}

	executorCfg := adapterjudge.ExecutorConfig{Languages: execLanguages}

	pool := judgepool.NewPool(poolCfg, dockerClient)
	pool.Start()
	defer pool.Stop()

	executor := adapterjudge.NewExecutor(pool, dockerClient, executorCfg)

	var sourceCodeDownloader appjudge.SourceCodeDownloader
	var testCaseProvider appjudge.TestCaseProvider
	var outputChecker appjudge.OutputChecker

	switch storageBackend {
	case "gcs":
		if gcsBucket == "" {
			slog.Error("worker: GCS_BUCKET is required when STORAGE_BACKEND=gcs")
			os.Exit(1)
		}
		gcsClient, err := googleStorage.NewClient(ctx)
		if err != nil {
			slog.Error("worker: failed to create gcs client", "error", err)
			os.Exit(1)
		}
		defer gcsClient.Close()
		sourceCodeDownloader = adapterjudge.NewSourceCodeDownloader(gcsClient, gcsBucket)
		testCaseProvider = adapterjudge.NewTestCaseProvider(gcsClient, gcsBucket, dbPool)
		outputChecker = adapterjudge.NewOutputComparator(gcsClient, gcsBucket)
		slog.Info("worker: using GCS storage backend", "bucket", gcsBucket)
	default:
		sourceCodeDownloader = adapterjudge.NewSourceCodeDownloaderLocal(storageLocalDir)
		testCaseProvider = adapterjudge.NewTestCaseProviderLocal(storageLocalDir, dbPool)
		outputChecker = adapterjudge.NewOutputComparatorLocal(storageLocalDir)
		slog.Info("worker: using local storage backend", "dir", storageLocalDir)
	}

	problemProvider := adapterjudge.NewProblemProvider(dbPool)
	submissionRepo := adaptersubmission.NewRepository(dbPool)
	submissionUpdater := adapterjudge.NewSubmissionUpdater(submissionRepo)

	// judge use case

	judgeSubmissionUseCase := appjudge.NewJudgeSubmissionUseCase(
		submissionUpdater,
		sourceCodeDownloader,
		problemProvider,
		testCaseProvider,
		executor,
		outputChecker,
		txManager,
	)

	// consume loop

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("worker: listening for submissions")

	if err := queue.Consume(ctx, func(ctx context.Context, msg appsubmission.SubmissionQueueMessage) error {
		if err := judgeSubmissionUseCase.Execute(ctx, appjudge.JudgeSubmissionInput{
			SubmissionID: msg.SubmissionID,
		}); err != nil {
			slog.ErrorContext(ctx, "worker: judge execution failed", "error", err, "submission_id", msg.SubmissionID)
		}
		return nil
	}); err != nil {
		slog.Error("worker: consume loop ended with error", "error", err)
		os.Exit(1)
	}

	slog.Info("worker: shutdown complete")
}

type judgeLanguageConfig struct {
	Image       string `yaml:"image"`
	CPU         string `yaml:"cpu"`
	MemoryBytes int64  `yaml:"memoryBytes"`
	CompileCmd  string `yaml:"compileCmd"`
	RunCmd      string `yaml:"runCmd"`
	Extension   string `yaml:"extension"`
}

type judgeSection struct {
	IdleTimeoutMinutes  int                            `yaml:"idleTimeoutMinutes"`
	MemoryOverheadBytes int64                          `yaml:"memoryOverheadBytes"`
	Languages           map[string]judgeLanguageConfig `yaml:"languages"`
}

type judgeConfigFile struct {
	Judge judgeSection `yaml:"judge"`
}

func loadJudgeConfig() judgeConfigFile {
	path := getEnv("JUDGE_CONFIG", "config/judge_config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Error("worker: failed to read judge config", "path", path, "error", err)
		os.Exit(1)
	}
	var cfg judgeConfigFile
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		slog.Error("worker: failed to parse judge config", "path", path, "error", err)
		os.Exit(1)
	}
	if len(cfg.Judge.Languages) == 0 {
		slog.Error("worker: judge config has no languages defined", "path", path)
		os.Exit(1)
	}
	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getRequiredEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("worker: required env var is not set", "key", key)
		os.Exit(1)
	}
	return v
}

func getRequiredEnvInt64(key string) int64 {
	s := getRequiredEnv(key)
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		slog.Error("worker: env var must be an integer", "key", key, "value", s)
		os.Exit(1)
	}
	return n
}
