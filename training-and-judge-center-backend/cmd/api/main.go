package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	googleStorage "cloud.google.com/go/storage"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/config"
	"github.com/training-judge-center/backend/internal/infrastructure/parser"
	infraPostgres "github.com/training-judge-center/backend/internal/infrastructure/postgres"
	infraSt "github.com/training-judge-center/backend/internal/infrastructure/storage"
	platformConfig "github.com/training-judge-center/backend/internal/platform/config"
	platformParser "github.com/training-judge-center/backend/internal/platform/parser"
	platformPostgres "github.com/training-judge-center/backend/internal/platform/postgres"
	platformUser "github.com/training-judge-center/backend/internal/platform/user"
	"github.com/training-judge-center/backend/internal/server"
	"github.com/training-judge-center/backend/internal/server/handler/problem"
)

func main() {
	cfg := config.Load()

	if cfg.VirtualObject == nil {
		slog.Error("virtual object config cannot be nil")
		os.Exit(1)
	}

	ctx := context.Background()

	dbPool, err := platformPostgres.NewConnectionPool(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()
	slog.Info("database connected successfully")

	// Repositories & Services
	problemRepo := infraPostgres.NewProblemRepository(dbPool)
	settingsProvider := platformConfig.NewVirtualObjectProvider(cfg.VirtualObject)

	// File Storage
	var fileStorage appProblem.ProblemFileRepository
	switch cfg.StorageBackend {
	case "gcs":
		if cfg.GCSBucket == "" {
			slog.Error("GCS_BUCKET env var is required when STORAGE_BACKEND=gcs")
			os.Exit(1)
		}
		gcsClient, err := googleStorage.NewClient(ctx)
		if err != nil {
			slog.Error("failed to create GCS client", "error", err)
			os.Exit(1)
		}
		slog.Info("using GCS storage backend", "bucket", cfg.GCSBucket)
		fileStorage = infraSt.NewGCSProblemFileRepository(gcsClient, cfg.GCSBucket)
	default:
		localDir := cfg.StorageLocalDir
		localRepo, err := infraSt.NewLocalStorageRepository(localDir)
		if err != nil {
			slog.Error("failed to create local file storage", "error", err)
			os.Exit(1)
		}
		slog.Info("using local storage backend", "dir", localDir)
		fileStorage = localRepo
	}

	icpcParser := parser.NewICPCParser(
		settingsProvider.GetMaxFileSizeTestCaseMB(),
		settingsProvider.GetMaxFileSizeDefaultMB(),
		settingsProvider.GetMaxFileCountTestCase(),
		settingsProvider.GetMaxFileCountSample(),
		cfg.VirtualObject.LanguageExtensions,
	)
	zipParserAdapter := platformParser.NewICPCParserAdapter(icpcParser)
	packageParserAdapter := platformParser.NewICPCPackageParserAdapter(icpcParser)

	var userProvider appProblem.UserProvider
	if cfg.MockAuth {
		slog.Info("running in MOCK_AUTH mode")
		userProvider = platformUser.NewMockDisplayProvider()
	} else {
		slog.Error("real UserProvider is not implemented yet; set MOCK_AUTH=1 for local development")
		os.Exit(1)
	}

	// Use Cases
	createProblemUseCase := appProblem.NewCreateProblemUseCase(problemRepo, settingsProvider)
	importProblemUseCase := appProblem.NewImportProblemUseCase(problemRepo, fileStorage, packageParserAdapter, settingsProvider)
	updateProblemUseCase := appProblem.NewUpdateProblemUseCase(problemRepo, settingsProvider)
	uploadProblemFilesUseCase := appProblem.NewUploadProblemFilesUseCase(problemRepo, fileStorage, zipParserAdapter, settingsProvider)
	deleteProblemFileUseCase := appProblem.NewDeleteProblemFileUseCase(problemRepo, fileStorage)
	addModifierUseCase := appProblem.NewAddModifierUseCase(problemRepo, userProvider)
	removeModifierUseCase := appProblem.NewRemoveModifierUseCase(problemRepo)
	listModifiersUseCase := appProblem.NewListModifiersUseCase(problemRepo)
	getProblemUseCase := appProblem.NewGetProblemUseCase(problemRepo, userProvider)
	listProblemsUseCase := appProblem.NewListProblemsUseCase(problemRepo, userProvider)
	unpublishProblemUseCase := appProblem.NewUnpublishProblemUseCase(problemRepo)
	changeAccessibilityUseCase := appProblem.NewChangeAccessibilityUseCase(problemRepo)
	deleteProblemUseCase := appProblem.NewDeleteProblemUseCase(problemRepo, fileStorage)

	problemHandler := problem.NewHandler(
		createProblemUseCase,
		importProblemUseCase,
		updateProblemUseCase,
		uploadProblemFilesUseCase,
		deleteProblemFileUseCase,
		addModifierUseCase,
		removeModifierUseCase,
		listModifiersUseCase,
		getProblemUseCase,
		listProblemsUseCase,
		unpublishProblemUseCase,
		changeAccessibilityUseCase,
		deleteProblemUseCase,
		userProvider,
		settingsProvider,
	)
	router := server.NewRouter(cfg, &server.Handlers{Problem: problemHandler})

	slog.Info("server starting", "port", cfg.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", cfg.Port), router); err != nil {
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}
