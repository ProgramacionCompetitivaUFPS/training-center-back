package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	googleStorage "cloud.google.com/go/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/moby/moby/client"
	"gopkg.in/yaml.v3"

	adapterjudge "github.com/training-judge-center/backend/internal/adapter/judge"
	judgepool "github.com/training-judge-center/backend/internal/adapter/judge/pool"
	infrapostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	adapterproblem "github.com/training-judge-center/backend/internal/adapter/problem"
	adapterqueue "github.com/training-judge-center/backend/internal/adapter/queue"
	adaptersubmission "github.com/training-judge-center/backend/internal/adapter/submission"
	appjudge "github.com/training-judge-center/backend/internal/application/judge"
	appproblem "github.com/training-judge-center/backend/internal/application/problem"
	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
)

// poolSolutions keys the pool that compiles and runs contestant code. It is a
// map key, so a typo would silently read a zero-valued pool instead of failing.
const poolSolutions = "solutions"

// Floors applied when the matching judge config key is missing or non-positive.
// KnownFields rejects unknown keys but says nothing about absent ones, so every
// optional value needs one.
const (
	defaultIdleTimeoutMinutes          = 10
	defaultDockerDaemonReserveBytes    = 512 << 20 // 512 MiB
	defaultDockerDaemonReserveCores    = 1
	defaultStaleRunningAfterMinutes    = 10
	defaultStaleValidationAfterMinutes = 20
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

	podCPUMillis := getRequiredEnvInt64("POD_CPU_LIMIT")
	maxConcurrent := int(podCPUMillis/1000) - judgeCfg.Judge.DockerDaemonReserveCores
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	slog.Info("worker: concurrency derived",
		"pod_cpu_millicores", podCPUMillis,
		"docker_daemon_reserve_cores", judgeCfg.Judge.DockerDaemonReserveCores,
		"max_concurrent", maxConcurrent)

	// Sizing and image now come from different sections, so they are joined
	// here: the pool decides how big a container is, the language which image
	// it runs.
	solutionSizing := judgeCfg.Judge.Pools[poolSolutions].Languages
	poolLanguages := make(map[string]judgepool.LanguageConfig, len(solutionSizing))
	for lang, sizing := range solutionSizing {
		poolLanguages[lang] = judgepool.LanguageConfig{
			Image:       judgeCfg.Judge.Languages[lang].Image,
			CPU:         sizing.CPU,
			MemoryBytes: sizing.MemoryBytes,
		}
	}

	execLanguages := make(map[string]adapterjudge.LanguageExecConfig, len(judgeCfg.Judge.Languages))
	artifactLanguages := make(map[string]adapterjudge.ArtifactLanguageConfig, len(judgeCfg.Judge.Languages))
	for lang, lc := range judgeCfg.Judge.Languages {
		execLanguages[lang] = adapterjudge.LanguageExecConfig{
			CompileCmd: lc.CompileCmd,
			RunCmd:     lc.RunCmd,
			Extension:  lc.Extension,
		}
		artifactLanguages[lang] = adapterjudge.ArtifactLanguageConfig{
			SourcePath:   lc.ArtifactSource,
			CompileCmd:   lc.ArtifactCompile,
			ArtifactPath: lc.ArtifactPath,
		}
	}

	poolCfg := judgepool.PoolConfig{
		MemLimitBytes: poolMemLimit,
		OverheadBytes: judgeCfg.Judge.DockerDaemonReserveBytes,
		IdleTimeout:   time.Duration(judgeCfg.Judge.IdleTimeoutMinutes) * time.Minute,
		ReapInterval:  time.Minute,
		Languages:     poolLanguages,
	}

	executorCfg := adapterjudge.ExecutorConfig{Languages: execLanguages}

	pool := judgepool.NewPool(poolCfg, dockerClient)
	pool.Start()
	defer pool.Stop()

	executor := adapterjudge.NewExecutor(pool, dockerClient, executorCfg)
	artifactCompiler := adapterjudge.NewArtifactCompiler(pool, dockerClient, adapterjudge.ArtifactCompilerConfig{Languages: artifactLanguages})

	var sourceCodeDownloader appjudge.SourceCodeDownloader
	var testCaseProvider appjudge.TestCaseProvider
	var outputChecker appjudge.OutputChecker
	var artifactUploader appjudge.ArtifactUploader

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
		artifactUploader = adapterjudge.NewArtifactUploader(gcsClient, gcsBucket)
		slog.Info("worker: using GCS storage backend", "bucket", gcsBucket)
	default:
		sourceCodeDownloader = adapterjudge.NewSourceCodeDownloaderLocal(storageLocalDir)
		testCaseProvider = adapterjudge.NewTestCaseProviderLocal(storageLocalDir, dbPool)
		outputChecker = adapterjudge.NewOutputComparatorLocal(storageLocalDir)
		artifactUploader = adapterjudge.NewArtifactUploaderLocal(storageLocalDir)
		slog.Info("worker: using local storage backend", "dir", storageLocalDir)
	}

	problemProvider := adapterjudge.NewProblemProvider(dbPool)
	solutionProvider := adapterjudge.NewSolutionProvider(dbPool)
	judgingSourceProvider := adapterjudge.NewJudgingSourceProvider(dbPool)
	validatorRunner := adapterjudge.NewValidatorRunner()
	submissionRepo := adaptersubmission.NewRepository(dbPool)
	submissionUpdater := adapterjudge.NewSubmissionUpdater(submissionRepo)
	staleSubmissionRecoverer := adapterjudge.NewStaleSubmissionRecoverer(dbPool)

	// judge use case

	judgeSubmissionUseCase := appjudge.NewJudgeSubmissionUseCase(
		submissionUpdater,
		sourceCodeDownloader,
		problemProvider,
		testCaseProvider,
		executor,
		outputChecker,
		txManager,
		appjudge.DefaultRetryConfig(),
	)

	recoverStaleSubmissionsUseCase := appjudge.NewRecoverStaleSubmissionsUseCase(
		staleSubmissionRecoverer,
		time.Duration(judgeCfg.Judge.StaleRunningAfterMinutes)*time.Minute,
	)

	// problem validation use case

	validateSolutionsUseCase := appjudge.NewValidateSolutionsUseCase(
		solutionProvider,
		sourceCodeDownloader,
		problemProvider,
		testCaseProvider,
		executor,
		outputChecker,
		appjudge.DefaultRetryConfig(),
	)
	solutionValidator := adapterproblem.NewSolutionValidator(validateSolutionsUseCase)

	prepareJudgingUseCase := appjudge.NewPrepareJudgingUseCase(
		judgingSourceProvider,
		sourceCodeDownloader,
		artifactCompiler,
		artifactUploader,
		testCaseProvider,
		validatorRunner,
	)
	judgingPreparer := adapterproblem.NewJudgingPreparer(prepareJudgingUseCase)
	judgingArtifactWriter := adapterproblem.NewJudgingArtifactWriter(dbPool)

	problemValidationRepo := adapterproblem.NewProblemValidationRepository(dbPool)
	problemPublisher := adapterproblem.NewProblemPublisher(dbPool)

	validateProblemUseCase := appproblem.NewValidateProblemUseCase(
		problemValidationRepo,
		judgingPreparer,
		judgingArtifactWriter,
		solutionValidator,
		problemPublisher,
		txManager,
	)

	staleValidationRecoverer := adapterproblem.NewStaleValidationRecoverer(dbPool)
	recoverStaleValidationsUseCase := appproblem.NewRecoverStaleValidationsUseCase(
		staleValidationRecoverer,
		time.Duration(judgeCfg.Judge.StaleValidationAfterMinutes)*time.Minute,
	)

	// background guardians

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				healthy := pool.IsHealthy(pingCtx)
				cancel()
				if !healthy {
					slog.Error("worker: container pool is unhealthy, exiting so the pod restarts")
					os.Exit(1)
				}
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				count, err := recoverStaleSubmissionsUseCase.Execute(ctx)
				if err == nil && count > 0 {
					slog.Info("worker: stale submissions recovered", "count", count)
				}
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				count, err := recoverStaleValidationsUseCase.Execute(ctx)
				if err == nil && count > 0 {
					slog.Info("worker: stale problem validations recovered", "count", count)
				}
			}
		}
	}()

	// consume loop

	slog.Info("worker: listening for submissions")

	if err := queue.Consume(ctx, maxConcurrent,
		adapterqueue.NewSubmissionPayloadHandler(func(ctx context.Context, msg appsubmission.SubmissionQueueMessage) error {
			if err := judgeSubmissionUseCase.Execute(ctx, appjudge.JudgeSubmissionInput{
				SubmissionID: msg.SubmissionID,
			}); err != nil {
				slog.ErrorContext(ctx, "worker: judge execution failed", "error", err, "submission_id", msg.SubmissionID)
			}
			return nil
		}),
		adapterqueue.NewValidationPayloadHandler(func(ctx context.Context, msg appproblem.ValidationQueueMessage) error {
			if err := validateProblemUseCase.Execute(ctx, appproblem.ValidateProblemInput{
				ValidationID: msg.ValidationID,
				Slug:         msg.Slug,
			}); err != nil {
				slog.ErrorContext(ctx, "worker: problem validation failed", "error", err, "validation_id", msg.ValidationID)
			}
			return nil
		}),
	); err != nil {
		slog.Error("worker: consume loop ended with error", "error", err)
		os.Exit(1)
	}

	slog.Info("worker: shutdown complete")
}

// judgeLanguageConfig holds what is true of a language regardless of which pool
// runs it: its image and how its code is compiled and executed.
type judgeLanguageConfig struct {
	Image           string `yaml:"image"`
	Extension       string `yaml:"extension"`
	CompileCmd      string `yaml:"compileCmd"`
	RunCmd          string `yaml:"runCmd"`
	ArtifactSource  string `yaml:"artifactSource"`
	ArtifactCompile string `yaml:"artifactCompile"`
	ArtifactPath    string `yaml:"artifactPath"`
	ArtifactRun     string `yaml:"artifactRun"`
}

// judgePoolLanguageConfig is the sizing of one language's containers inside one
// pool. The same language can need different limits depending on what the
// container is used for, so this lives under the pool, not under the language.
type judgePoolLanguageConfig struct {
	CPU         string `yaml:"cpu"`
	MemoryBytes int64  `yaml:"memoryBytes"`
}

type judgePoolConfig struct {
	Languages map[string]judgePoolLanguageConfig `yaml:"languages"`
}

type judgeSection struct {
	IdleTimeoutMinutes int `yaml:"idleTimeoutMinutes"`
	// Both reserves are carved out of the dind container's limits, where the
	// Docker daemon and the containers it spawns live. Neither has anything to
	// do with the worker process, which runs in a separate container with its
	// own cgroup and its own limits.
	DockerDaemonReserveBytes    int64                          `yaml:"dockerDaemonReserveBytes"`
	DockerDaemonReserveCores    int                            `yaml:"dockerDaemonReserveCores"`
	StaleRunningAfterMinutes    int                            `yaml:"staleRunningAfterMinutes"`
	StaleValidationAfterMinutes int                            `yaml:"staleValidationAfterMinutes"`
	Languages                   map[string]judgeLanguageConfig `yaml:"languages"`
	Pools                       map[string]judgePoolConfig     `yaml:"pools"`
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
	// Strict: an unknown key means the file and these structs have drifted, and
	// a silently ignored key would show up much later as a zero-valued limit.
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		slog.Error("worker: failed to parse judge config", "path", path, "error", err)
		os.Exit(1)
	}
	if err := validateJudgeConfig(cfg); err != nil {
		slog.Error("worker: invalid judge config", "path", path, "error", err)
		os.Exit(1)
	}
	applyJudgeConfigDefaults(&cfg)
	return cfg
}

// validateJudgeConfig reports the first inconsistency found. It is separate from
// loadJudgeConfig, which exits the process, so it can be tested.
func validateJudgeConfig(cfg judgeConfigFile) error {
	if len(cfg.Judge.Languages) == 0 {
		return errors.New("no languages defined")
	}
	if _, ok := cfg.Judge.Pools[poolSolutions]; !ok {
		return fmt.Errorf("no %q pool defined", poolSolutions)
	}
	// A pool sizing a language with no image would only fail when that language
	// is first claimed, mid-judging.
	for poolName, pool := range cfg.Judge.Pools {
		if len(pool.Languages) == 0 {
			return fmt.Errorf("pool %q sizes no languages", poolName)
		}
		for lang, sizing := range pool.Languages {
			if _, ok := cfg.Judge.Languages[lang]; !ok {
				return fmt.Errorf("pool %q sizes undeclared language %q", poolName, lang)
			}
			// An empty cpu parses as no limit at all, which silently breaks the
			// one-CPU-per-container assumption the concurrency formula rests on.
			if sizing.CPU == "" {
				return fmt.Errorf("pool %q, language %q: no cpu", poolName, lang)
			}
			if sizing.MemoryBytes <= 0 {
				return fmt.Errorf("pool %q, language %q: memoryBytes is not positive", poolName, lang)
			}
		}
	}
	for lang, lc := range cfg.Judge.Languages {
		if lc.Image == "" {
			return fmt.Errorf("language %q has no image", lang)
		}
		// A language with a runCmd is one solutions are written in, so it must
		// be sized by the solutions pool and able to build a checker or
		// validator. Entries without one are not languages code is sent in.
		if lc.RunCmd == "" {
			continue
		}
		if lc.Extension == "" {
			return fmt.Errorf("language %q runs solutions but has no extension", lang)
		}
		if _, ok := cfg.Judge.Pools[poolSolutions].Languages[lang]; !ok {
			return fmt.Errorf("language %q runs solutions but pool %q does not size it", lang, poolSolutions)
		}
		for _, f := range []struct{ name, value string }{
			{"artifactSource", lc.ArtifactSource},
			{"artifactCompile", lc.ArtifactCompile},
			{"artifactPath", lc.ArtifactPath},
			{"artifactRun", lc.ArtifactRun},
		} {
			if f.value == "" {
				return fmt.Errorf("language %q: %s is empty", lang, f.name)
			}
			if !strings.Contains(f.value, adapterjudge.ArtifactNamePlaceholder) {
				return fmt.Errorf("language %q: %s has no %s", lang, f.name, adapterjudge.ArtifactNamePlaceholder)
			}
		}
	}
	return nil
}

// applyJudgeConfigDefaults floors the optional values. KnownFields rejects
// unknown keys but says nothing about absent ones.
func applyJudgeConfigDefaults(cfg *judgeConfigFile) {
	if cfg.Judge.IdleTimeoutMinutes <= 0 {
		cfg.Judge.IdleTimeoutMinutes = defaultIdleTimeoutMinutes
	}
	if cfg.Judge.DockerDaemonReserveBytes <= 0 {
		cfg.Judge.DockerDaemonReserveBytes = defaultDockerDaemonReserveBytes
	}
	if cfg.Judge.DockerDaemonReserveCores <= 0 {
		cfg.Judge.DockerDaemonReserveCores = defaultDockerDaemonReserveCores
	}
	if cfg.Judge.StaleRunningAfterMinutes <= 0 {
		cfg.Judge.StaleRunningAfterMinutes = defaultStaleRunningAfterMinutes
	}
	if cfg.Judge.StaleValidationAfterMinutes <= 0 {
		cfg.Judge.StaleValidationAfterMinutes = defaultStaleValidationAfterMinutes
	}
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
