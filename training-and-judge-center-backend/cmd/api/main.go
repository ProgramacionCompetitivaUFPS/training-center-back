package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	googleStorage "cloud.google.com/go/storage"
	"google.golang.org/api/option"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/config"
	"github.com/training-judge-center/backend/internal/infrastructure/parser"
	infraSt "github.com/training-judge-center/backend/internal/infrastructure/storage"
	platformConfig "github.com/training-judge-center/backend/internal/platform/config"
	"github.com/training-judge-center/backend/internal/platform/postgres"
	platformUser "github.com/training-judge-center/backend/internal/platform/user"
	"github.com/training-judge-center/backend/internal/server"
	"github.com/training-judge-center/backend/internal/server/handler/problem"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	dbPool, err := postgres.NewConnectionPool(ctx, cfg)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()
	slog.Info("database connected successfully")

	// Repositories & Services
	problemRepo := postgres.NewProblemRepository(dbPool)
	settingsProvider := platformConfig.NewVirtualObjectProvider(cfg.VirtualObject)

	// File Storage
	var fileStorage appProblem.ProblemFileRepository
	switch cfg.StorageBackend {
	case "gcs":
		if cfg.GCSBucket == "" {
			slog.Error("GCS_BUCKET env var is required when STORAGE_BACKEND=gcs")
			os.Exit(1)
		}
		gcsClient, err := googleStorage.NewClient(ctx, option.WithoutAuthentication())
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
		settingsProvider.GetMaxFileCountTestCase(),
		settingsProvider.GetMaxFileCountSample(),
	)

	var userProvider appProblem.UserProvider
	if cfg.MockAuth {
		slog.Info("running in MOCK_AUTH mode")
		userProvider = platformUser.NewMockDisplayProvider()
	} else {
		// TODO: userProvider = postgres.NewUserRepository(db)
		userProvider = platformUser.NewMockDisplayProvider()
	}

	// Use Cases
	createProblemUseCase := appProblem.NewCreateProblemUseCase(problemRepo, settingsProvider)
	updateProblemUseCase := appProblem.NewUpdateProblemUseCase(problemRepo, settingsProvider)
	uploadProblemFilesUseCase := appProblem.NewUploadProblemFilesUseCase(problemRepo, fileStorage, icpcParser, settingsProvider)
	deleteProblemFileUseCase := appProblem.NewDeleteProblemFileUseCase(problemRepo, fileStorage)
	addModifierUseCase := appProblem.NewAddModifierUseCase(problemRepo, userProvider)
	removeModifierUseCase := appProblem.NewRemoveModifierUseCase(problemRepo)
	listModifiersUseCase := appProblem.NewListModifiersUseCase(problemRepo)

	problemHandler := problem.NewHandler(
		createProblemUseCase,
		updateProblemUseCase,
		uploadProblemFilesUseCase,
		deleteProblemFileUseCase,
		addModifierUseCase,
		removeModifierUseCase,
		listModifiersUseCase,
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
