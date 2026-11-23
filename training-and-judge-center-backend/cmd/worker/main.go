package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	googleStorage "cloud.google.com/go/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/moby/moby/client"

	adapterjudge "github.com/training-judge-center/backend/internal/adapter/judge"
	judgepool "github.com/training-judge-center/backend/internal/adapter/judge/pool"
	infrapostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	adapterproblem "github.com/training-judge-center/backend/internal/adapter/problem"
	adapterqueue "github.com/training-judge-center/backend/internal/adapter/queue"
	adaptersubmission "github.com/training-judge-center/backend/internal/adapter/submission"
	appjudge "github.com/training-judge-center/backend/internal/application/judge"
	appproblem "github.com/training-judge-center/backend/internal/application/problem"
	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
	"github.com/training-judge-center/backend/internal/config"
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

// rabbitmqConnectTimeout covers the broker accepting AMQP after it reports healthy.
const rabbitmqConnectTimeout = time.Minute

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
	queue, err := adapterqueue.ConnectRabbitMQQueue(rabbitmqURL, rabbitmqConnectTimeout)
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

	testData := testDataConfig(judgeCfg, config.LoadVirtualObject())
	if err := validateTestDataBudget(testData, maxConcurrent); err != nil {
		slog.Error("worker: test data budget does not fit the limits", "error", err)
		os.Exit(1)
	}

	execLanguages := make(map[string]adapterjudge.LanguageExecConfig, len(judgeCfg.Judge.Languages))
	artifactLanguages := make(map[string]adapterjudge.ArtifactLanguageConfig, len(judgeCfg.Judge.Languages))
	for lang, lc := range judgeCfg.Judge.Languages {
		execLanguages[lang] = adapterjudge.LanguageExecConfig{
			CompileCmd:   lc.CompileCmd,
			RunCmd:       lc.RunCmd,
			Extension:    lc.Extension,
			MemoryFactor: lc.MemoryFactor,
		}
		artifactLanguages[lang] = adapterjudge.ArtifactLanguageConfig{
			SourcePath:   lc.ArtifactSource,
			CompileCmd:   lc.ArtifactCompile,
			ArtifactPath: lc.ArtifactPath,
			RunCmd:       lc.ArtifactRun,
		}
	}

	executorCfg := adapterjudge.ExecutorConfig{Languages: execLanguages}

	volumeSource := getRequiredEnv("JUDGE_VOLUME_SOURCE")
	if err := ensureSharedVolume(judgepool.SharedVolumePath); err != nil {
		slog.Error("worker: the judging volume is not usable", "error", err)
		os.Exit(1)
	}

	idleTimeout := time.Duration(judgeCfg.Judge.IdleTimeoutMinutes) * time.Minute

	// Assumes the Docker daemon is exclusive to this worker (a dind sidecar per pod, or the single compose stack): another worker sharing it would lose its live sandboxes here.
	// Before any pool exists, so whatever carries the label belongs to a worker that is gone.
	cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), 30*time.Second)
	removed, err := judgepool.RemoveOrphans(cleanupCtx, dockerClient)
	cancelCleanup()
	if err != nil {
		slog.Error("worker: could not clean up leftover sandboxes", "error", err)
		os.Exit(1)
	}
	slog.Info("worker: leftover sandboxes removed", "count", removed)

	heavyPool := judgepool.NewPool(poolConfigFor(judgeCfg, poolHeavy, idleTimeout, volumeSource), dockerClient)
	heavyPool.Start()
	defer heavyPool.Stop()

	lightPool := judgepool.NewPool(poolConfigFor(judgeCfg, poolLight, idleTimeout, volumeSource), dockerClient)
	lightPool.Start()
	defer lightPool.Stop()

	// Before anything is judged: a mount whose source the daemon resolves
	// elsewhere is silent, and would turn every submission into a wrong answer.
	probeCtx, cancelProbe := context.WithTimeout(context.Background(), 30*time.Second)
	if err := adapterjudge.VerifySharedVolume(probeCtx, lightPool, dockerClient, judgepool.SharedVolumePath); err != nil {
		cancelProbe()
		slog.Error("worker: the judging volume is not shared with the sandboxes", "error", err)
		os.Exit(1)
	}
	cancelProbe()
	slog.Info("worker: judging volume verified end to end")

	executor := adapterjudge.NewExecutor(heavyPool, dockerClient, executorCfg, judgepool.SharedVolumePath)
	artifactCfg := adapterjudge.ArtifactConfig{Languages: artifactLanguages}
	artifactCompiler := adapterjudge.NewArtifactCompiler(heavyPool, dockerClient, artifactCfg)

	var sourceCodeDownloader appjudge.SourceCodeDownloader
	var testCaseProvider appjudge.TestCaseProvider
	var outputChecker appjudge.OutputChecker
	var artifactUploader appjudge.ArtifactUploader
	var validatorRunner appjudge.ValidatorRunner
	var stagedFileCleaner appproblem.StagedFileCleaner

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
		testCaseProvider, err = adapterjudge.NewTestCaseProvider(gcsClient, gcsBucket, dbPool, testData)
		if err != nil {
			slog.Error("worker: failed to prepare the test data cache", "error", err)
			os.Exit(1)
		}
		outputChecker = adapterjudge.NewOutputChecker(lightPool, dockerClient, artifactCfg, gcsClient, gcsBucket, judgepool.SharedVolumePath)
		artifactUploader = adapterjudge.NewArtifactUploader(gcsClient, gcsBucket)
		validatorRunner = adapterjudge.NewValidatorRunner(lightPool, dockerClient, artifactCfg, gcsClient, gcsBucket)
		stagedFileCleaner = adapterproblem.NewGCSFileRepository(gcsClient, gcsBucket)
		slog.Info("worker: using GCS storage backend", "bucket", gcsBucket)
	default:
		sourceCodeDownloader = adapterjudge.NewSourceCodeDownloaderLocal(storageLocalDir)
		testCaseProvider, err = adapterjudge.NewTestCaseProviderLocal(storageLocalDir, dbPool, testData)
		if err != nil {
			slog.Error("worker: failed to prepare the test data cache", "error", err)
			os.Exit(1)
		}
		outputChecker = adapterjudge.NewOutputCheckerLocal(lightPool, dockerClient, artifactCfg, storageLocalDir, judgepool.SharedVolumePath)
		artifactUploader = adapterjudge.NewArtifactUploaderLocal(storageLocalDir)
		validatorRunner = adapterjudge.NewValidatorRunnerLocal(lightPool, dockerClient, artifactCfg, storageLocalDir)
		localFiles, err := adapterproblem.NewLocalFileRepository(storageLocalDir)
		if err != nil {
			slog.Error("worker: failed to create local file storage", "error", err)
			os.Exit(1)
		}
		stagedFileCleaner = localFiles
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

	trackedSubmissionQueue := adaptersubmission.NewTrackedQueue(dbPool, adapterqueue.NewRabbitMQSubmissionQueue(queue))
	recoverStalePendingSubmissionsUseCase := appjudge.NewRecoverStalePendingSubmissionsUseCase(
		adaptersubmission.NewStalePendingRecoverer(dbPool, trackedSubmissionQueue),
		time.Duration(judgeCfg.Judge.StalePendingAfterMinutes)*time.Minute,
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
	problemRepo := adapterproblem.NewRepository(dbPool)

	validateProblemUseCase := appproblem.NewValidateProblemUseCase(
		problemValidationRepo,
		judgingPreparer,
		judgingArtifactWriter,
		solutionValidator,
		problemPublisher,
		txManager,
		problemRepo,
		stagedFileCleaner,
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
				requeued, failed, err := recoverStalePendingSubmissionsUseCase.Execute(ctx)
				if err == nil && (requeued > 0 || failed > 0) {
					slog.Info("worker: stale pending submissions recovered", "requeued", requeued, "failed", failed)
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
				EnqueuedAt:   msg.EnqueuedAt,
			}); err != nil {
				slog.ErrorContext(ctx, "worker: judge execution failed", "error", err, "submission_id", msg.SubmissionID)
				// Acked either way; if it never started, let the recoverer re-publish it.
				trackedSubmissionQueue.Unqueue(ctx, msg.SubmissionID)
			}
			return nil
		}),
		adapterqueue.NewValidationPayloadHandler(func(ctx context.Context, msg appproblem.ValidationQueueMessage) error {
			if err := validateProblemUseCase.Execute(ctx, appproblem.ValidateProblemInput{
				ValidationID: msg.ValidationID,
				EnqueuedAt:   msg.EnqueuedAt,
				Slug:         msg.Slug,
			}); err != nil {
				slog.ErrorContext(ctx, "worker: problem validation failed", "error", err, "validation_id", msg.ValidationID)
			}
			return nil
		}),
	); err != nil {
		slog.Error("worker: consume loop ended with error", "error", err)
		// os.Exit skips the defers, and the sandboxes would outlive the worker.
		lightPool.Stop()
		heavyPool.Stop()
		os.Exit(1)
	}

	slog.Info("worker: shutdown complete")
}
