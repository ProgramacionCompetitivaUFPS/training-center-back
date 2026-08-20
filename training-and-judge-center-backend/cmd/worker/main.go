package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"os/signal"
	"slices"
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

// Pool names, which are also the keys of judge.pools. They are map keys, so a
// typo would silently read a zero-valued pool instead of failing.
const (
	poolHeavy = "heavy" // compiles every language, and runs contestant solutions
	poolLight = "light" // runs already-compiled checkers and validators
)

// reapInterval is how often the pool looks for containers idle past their
// timeout; unlike the timeout itself it is not worth configuring.
const reapInterval = time.Minute

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

	if err := validatePoolBudgets(judgeCfg, poolMemLimit, maxConcurrent); err != nil {
		slog.Error("worker: judge config does not fit this machine", "error", err)
		os.Exit(1)
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
			RunCmd:       lc.ArtifactRun,
		}
	}

	executorCfg := adapterjudge.ExecutorConfig{Languages: execLanguages}

	idleTimeout := time.Duration(judgeCfg.Judge.IdleTimeoutMinutes) * time.Minute

	heavyPool := judgepool.NewPool(poolConfigFor(judgeCfg, poolHeavy, idleTimeout), dockerClient)
	heavyPool.Start()
	defer heavyPool.Stop()

	lightPool := judgepool.NewPool(poolConfigFor(judgeCfg, poolLight, idleTimeout), dockerClient)
	lightPool.Start()
	defer lightPool.Stop()

	executor := adapterjudge.NewExecutor(heavyPool, dockerClient, executorCfg)
	artifactCfg := adapterjudge.ArtifactConfig{Languages: artifactLanguages}
	artifactCompiler := adapterjudge.NewArtifactCompiler(heavyPool, dockerClient, artifactCfg)

	var sourceCodeDownloader appjudge.SourceCodeDownloader
	var testCaseProvider appjudge.TestCaseProvider
	var outputChecker appjudge.OutputChecker
	var artifactUploader appjudge.ArtifactUploader
	var validatorRunner appjudge.ValidatorRunner

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
		validatorRunner = adapterjudge.NewValidatorRunner(lightPool, dockerClient, artifactCfg, gcsClient, gcsBucket)
		slog.Info("worker: using GCS storage backend", "bucket", gcsBucket)
	default:
		sourceCodeDownloader = adapterjudge.NewSourceCodeDownloaderLocal(storageLocalDir)
		testCaseProvider = adapterjudge.NewTestCaseProviderLocal(storageLocalDir, dbPool)
		outputChecker = adapterjudge.NewOutputComparatorLocal(storageLocalDir)
		artifactUploader = adapterjudge.NewArtifactUploaderLocal(storageLocalDir)
		validatorRunner = adapterjudge.NewValidatorRunnerLocal(lightPool, dockerClient, artifactCfg, storageLocalDir)
		slog.Info("worker: using local storage backend", "dir", storageLocalDir)
	}

	problemProvider := adapterjudge.NewProblemProvider(dbPool)
	solutionProvider := adapterjudge.NewSolutionProvider(dbPool)
	judgingSourceProvider := adapterjudge.NewJudgingSourceProvider(dbPool)
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
				// Both pools talk to the same daemon, so one ping answers for both.
				healthy := heavyPool.IsHealthy(pingCtx)
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
	BudgetBytes int64                              `yaml:"budgetBytes"`
	Languages   map[string]judgePoolLanguageConfig `yaml:"languages"`
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
	for _, name := range []string{poolHeavy, poolLight} {
		if _, ok := cfg.Judge.Pools[name]; !ok {
			return fmt.Errorf("no %q pool defined", name)
		}
	}
	// A pool sizing a language with no image would only fail when that language
	// is first claimed, mid-judging.
	for poolName, pool := range cfg.Judge.Pools {
		if pool.BudgetBytes <= 0 {
			return fmt.Errorf("pool %q: budgetBytes is not positive", poolName)
		}
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
		// be sized by the heavy pool and able to build a checker or validator.
		// Entries without one are not languages code is sent in.
		if lc.RunCmd == "" {
			continue
		}
		if lc.Extension == "" {
			return fmt.Errorf("language %q runs solutions but has no extension", lang)
		}
		if _, ok := cfg.Judge.Pools[poolHeavy].Languages[lang]; !ok {
			return fmt.Errorf("language %q runs solutions but pool %q does not size it", lang, poolHeavy)
		}
		// The artifact fields below mean a checker or validator can be written
		// in this language, and those run in the light pool.
		if _, ok := cfg.Judge.Pools[poolLight].Languages[lang]; !ok {
			return fmt.Errorf("language %q can carry a checker but pool %q does not size it", lang, poolLight)
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

// validatePoolBudgets checks the config against the machine it runs on. Its
// numbers come from the environment rather than the file, which is why it is
// separate from validateJudgeConfig.
// poolConfigFor joins the two config sections a pool needs: the pool says how
// big a container is, the language which image it runs.
func poolConfigFor(cfg judgeConfigFile, poolName string, idleTimeout time.Duration) judgepool.PoolConfig {
	pool := cfg.Judge.Pools[poolName]
	languages := make(map[string]judgepool.LanguageConfig, len(pool.Languages))
	for lang, sizing := range pool.Languages {
		languages[lang] = judgepool.LanguageConfig{
			Image:       cfg.Judge.Languages[lang].Image,
			CPU:         sizing.CPU,
			MemoryBytes: sizing.MemoryBytes,
		}
	}
	return judgepool.PoolConfig{
		BudgetBytes:  pool.BudgetBytes,
		IdleTimeout:  idleTimeout,
		ReapInterval: reapInterval,
		Languages:    languages,
	}
}

func validatePoolBudgets(cfg judgeConfigFile, dindMemBytes int64, maxConcurrent int) error {
	var total int64
	for _, pool := range cfg.Judge.Pools {
		total += pool.BudgetBytes
	}
	total += cfg.Judge.DockerDaemonReserveBytes
	if total > dindMemBytes {
		return fmt.Errorf("pool budgets plus the daemon reserve add up to %d bytes, over the %d the dind container has",
			total, dindMemBytes)
	}
	// D13: every judging holds one container in each pool, so a pool that
	// cannot fit maxConcurrent of its largest language blocks forever once they
	// are all busy — a silent hang instead of a loud failure.
	for _, name := range slices.Sorted(maps.Keys(cfg.Judge.Pools)) {
		pool := cfg.Judge.Pools[name]
		var largest int64
		for _, sizing := range pool.Languages {
			largest = max(largest, sizing.MemoryBytes)
		}
		if need := int64(maxConcurrent) * largest; pool.BudgetBytes < need {
			return fmt.Errorf("pool %q: budget %d is under the %d needed for %d concurrent containers of its largest language (%d bytes)",
				name, pool.BudgetBytes, need, maxConcurrent, largest)
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
