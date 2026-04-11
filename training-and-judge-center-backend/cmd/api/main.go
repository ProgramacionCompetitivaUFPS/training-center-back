package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	googleStorage "cloud.google.com/go/storage"
	"github.com/redis/go-redis/v9"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/config"
	"github.com/training-judge-center/backend/internal/infrastructure/parser"
	infraPostgres "github.com/training-judge-center/backend/internal/infrastructure/postgres"
	infraSt "github.com/training-judge-center/backend/internal/infrastructure/storage"
	"github.com/training-judge-center/backend/internal/platform/email"
	jwtplatform "github.com/training-judge-center/backend/internal/platform/jwt"
	platformConfig "github.com/training-judge-center/backend/internal/platform/config"
	platformParser "github.com/training-judge-center/backend/internal/platform/parser"
	platformPostgres "github.com/training-judge-center/backend/internal/platform/postgres"
	"github.com/training-judge-center/backend/internal/platform/ratelimit"
	redisplatform "github.com/training-judge-center/backend/internal/platform/redis"
	platformUser "github.com/training-judge-center/backend/internal/platform/user"
	"github.com/training-judge-center/backend/internal/server"
	"github.com/training-judge-center/backend/internal/server/handler"
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

	// Redis client
	redisOpt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		slog.Error("invalid redis URL", "error", err)
		os.Exit(1)
	}

	redisClient := redis.NewClient(redisOpt)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()
	slog.Info("redis connected successfully")

	// Problem repositories & settings
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

	// Problem use cases
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

	// User platform adapters
	userRepo := platformPostgres.NewUserRepository(dbPool)
	passwordRecoveryRepo := platformPostgres.NewPasswordRecoveryRepository(dbPool)
	emailChangeRepo := platformPostgres.NewEmailChangeRepository(dbPool)
	deactRepo := platformPostgres.NewDeactivationRequestRepository(dbPool)
	auditRepo := platformPostgres.NewDeactivationAuditLogRepository(dbPool)
	txManager := platformPostgres.NewPostgresTransactionManager(dbPool)
	jwtService := jwtplatform.NewService(cfg.JWTSecret, cfg.JWTExpirationHours)
	emailSender := email.NewSMTPSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom)
	redisRateLimiter := ratelimit.NewRedisRateLimiter(redisClient)
	sessionInvalidator := redisplatform.NewSessionInvalidator(redisClient, time.Duration(cfg.JWTExpirationHours)*time.Hour)

	// User use cases
	createUserUC := appuser.NewCreateUserUseCase(userRepo)
	loginUC := appuser.NewLoginUseCase(userRepo, jwtService)
	getUserProfileUC := appuser.NewGetUserProfileUseCase(userRepo)
	updateUserUC := appuser.NewUpdateUserUseCase(userRepo)
	updatePasswordUC := appuser.NewUpdatePasswordUseCase(userRepo, emailSender, sessionInvalidator, redisRateLimiter)
	adminUpdateUserUC := appuser.NewAdminUpdateUserUseCase(userRepo)
	adminDeactivateUserUC := appuser.NewAdminDeactivateUserUseCase(userRepo, sessionInvalidator)
	listUsersUC := appuser.NewListUsersUseCase(userRepo)
	requestEmailChangeUC := appuser.NewRequestEmailChangeUseCase(userRepo, emailChangeRepo, emailSender, redisRateLimiter)
	confirmEmailChangeUC := appuser.NewConfirmEmailChangeUseCase(userRepo, emailChangeRepo, emailSender, txManager)
	requestPasswordRecoveryUC := appuser.NewRequestPasswordRecoveryUseCase(userRepo, passwordRecoveryRepo, emailSender, redisRateLimiter)
	resetPasswordUC := appuser.NewResetPasswordUseCase(userRepo, passwordRecoveryRepo, sessionInvalidator, txManager)
	requestDeactUC := appuser.NewRequestDeactivationUseCase(userRepo, deactRepo, emailSender)
	confirmDeactUC := appuser.NewConfirmDeactivationUseCase(userRepo, deactRepo, auditRepo, emailSender, sessionInvalidator, txManager)

	// Handlers
	userHandler := handler.NewUserHandler(createUserUC, getUserProfileUC, updateUserUC, updatePasswordUC, adminUpdateUserUC, adminDeactivateUserUC, listUsersUC, requestEmailChangeUC, confirmEmailChangeUC, requestPasswordRecoveryUC, resetPasswordUC, requestDeactUC, confirmDeactUC)
	authHandler := handler.NewAuthHandler(loginUC)

	router := server.NewRouter(&server.Handlers{
		Problem: problemHandler,
		User:    userHandler,
		Auth:    authHandler,
	}, &server.Services{
		TokenService:       jwtService,
		SessionInvalidator: sessionInvalidator,
	})

	slog.Info("server starting", "port", cfg.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", cfg.Port), router); err != nil {
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}
