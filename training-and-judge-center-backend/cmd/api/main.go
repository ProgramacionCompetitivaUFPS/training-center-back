// @title           Training & Judge Center API
// @version         1.0
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

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
	"github.com/training-judge-center/backend/internal/adapter/auth"
	platformConfig "github.com/training-judge-center/backend/internal/adapter/config"
	"github.com/training-judge-center/backend/internal/adapter/email"
	"github.com/training-judge-center/backend/internal/adapter/group"
	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	handlerGroup "github.com/training-judge-center/backend/internal/adapter/http/handler/group"
	handlerMaterial "github.com/training-judge-center/backend/internal/adapter/http/handler/material"
	handlerProblem "github.com/training-judge-center/backend/internal/adapter/http/handler/problem"
	handlerUser "github.com/training-judge-center/backend/internal/adapter/http/handler/user"
	"github.com/training-judge-center/backend/internal/adapter/material"
	"github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/internal/adapter/problem"
	"github.com/training-judge-center/backend/internal/adapter/ratelimit"
	"github.com/training-judge-center/backend/internal/adapter/user"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/config"
	adapterhttp "github.com/training-judge-center/backend/internal/adapter/http"
)

func main() {
	cfg := config.Load()

	if cfg.VirtualObject == nil {
		slog.Error("virtual object config cannot be nil")
		os.Exit(1)
	}

	ctx := context.Background()

	dbPool, err := postgres.NewConnectionPool(ctx, cfg)
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
	problemRepo := problem.NewProblemRepository(dbPool)
	settingsProvider, err := platformConfig.NewPlatformSettings(cfg.VirtualObject)
	if err != nil {
		slog.Error("invalid platform settings in config", "error", err)
		os.Exit(1)
	}

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
		fileStorage = problem.NewGCSProblemFileRepository(gcsClient, cfg.GCSBucket)
	default:
		localDir := cfg.StorageLocalDir
		localRepo, err := problem.NewLocalStorageRepository(localDir)
		if err != nil {
			slog.Error("failed to create local file storage", "error", err)
			os.Exit(1)
		}
		slog.Info("using local storage backend", "dir", localDir)
		fileStorage = localRepo
	}

	icpcParser := problem.NewICPCParser(
		settingsProvider.MaxFileSizeTestCaseMB(),
		settingsProvider.MaxFileSizeDefaultMB(),
		settingsProvider.MaxFileCountTestCase(),
		settingsProvider.MaxFileCountSample(),
		cfg.VirtualObject.LanguageExtensions,
	)
	zipParserAdapter := problem.NewICPCParserAdapter(icpcParser)
	packageParserAdapter := problem.NewICPCPackageParserAdapter(icpcParser)

	userProvider := problem.NewProblemUserProvider(dbPool)

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

	problemHandler := handlerProblem.NewHandler(
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
	userRepo := user.NewUserRepository(dbPool)
	passwordRecoveryRepo := user.NewPasswordRecoveryRepository(dbPool)
	emailChangeRepo := user.NewEmailChangeRepository(dbPool)
	deactRepo := user.NewDeactivationRequestRepository(dbPool)
	auditRepo := user.NewDeactivationAuditLogRepository(dbPool)

	// Infrastructure and cross-cutting services
	txManager := postgres.NewPostgresTransactionManager(dbPool)
	jwtService := auth.NewJWTService(cfg.JWTSecret, cfg.JWTExpirationHours)
	emailSender := email.NewSMTPSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom)
	redisRateLimiter := ratelimit.NewRedisRateLimiter(redisClient)
	sessionInvalidator := auth.NewSessionInvalidator(redisClient, time.Duration(cfg.JWTExpirationHours)*time.Hour)

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
	userHandler := handlerUser.NewUserHandler(createUserUC, getUserProfileUC, updateUserUC, updatePasswordUC, adminUpdateUserUC, adminDeactivateUserUC, listUsersUC, requestEmailChangeUC, confirmEmailChangeUC, requestPasswordRecoveryUC, resetPasswordUC, requestDeactUC, confirmDeactUC)
	authHandler := handler.NewAuthHandler(loginUC)

	// Group repositories & platform adapters
	groupRepo := group.NewGroupRepository(dbPool)
	groupMemberRepo := group.NewMemberRepository(dbPool)
	groupUserProvider := group.NewUserProvider(dbPool)
	groupPrefsReader := group.NewPreferencesReader(dbPool)
	joinRequestRepo := group.NewJoinRequestRepository(dbPool)
	groupTxManager := postgres.NewPostgresTransactionManager(dbPool)

	// Group use cases
	createGroupUseCase := appGroup.NewCreateGroupUseCase(groupRepo, groupMemberRepo, groupTxManager)
	listGroupsUseCase := appGroup.NewListGroupsUseCase(groupRepo, groupMemberRepo)
	getGroupUseCase := appGroup.NewGetGroupUseCase(groupRepo, groupMemberRepo, groupUserProvider)
	listMyGroupsUseCase := appGroup.NewListMyGroupsUseCase(groupRepo, groupMemberRepo, groupPrefsReader)
	joinGroupUseCase := appGroup.NewJoinGroupUseCase(groupRepo, groupMemberRepo)
	requestJoinUseCase := appGroup.NewRequestJoinUseCase(groupRepo, groupMemberRepo, joinRequestRepo)
	approveRequestUseCase := appGroup.NewApproveRequestUseCase(groupMemberRepo, joinRequestRepo, groupTxManager)
	rejectRequestUseCase := appGroup.NewRejectRequestUseCase(groupMemberRepo, joinRequestRepo)
	listJoinRequestsUseCase := appGroup.NewListJoinRequestsUseCase(groupMemberRepo, joinRequestRepo, groupUserProvider)
	getMyRequestUseCase := appGroup.NewGetMyRequestUseCase(joinRequestRepo)
	cancelMyRequestUseCase := appGroup.NewCancelMyRequestUseCase(joinRequestRepo)

	groupInvitationJWTSvc := auth.NewGroupInvitationJWTService(cfg.JWTSecret)
	generateInviteUseCase := appGroup.NewGenerateInviteUseCase(groupRepo, groupMemberRepo, groupInvitationJWTSvc)
	acceptInviteUseCase := appGroup.NewAcceptInviteUseCase(groupRepo, groupMemberRepo, groupInvitationJWTSvc)

	groupHandler := handlerGroup.NewHandler(
		createGroupUseCase, listGroupsUseCase, getGroupUseCase, listMyGroupsUseCase,
		joinGroupUseCase,
		requestJoinUseCase, approveRequestUseCase, rejectRequestUseCase,
		listJoinRequestsUseCase, getMyRequestUseCase, cancelMyRequestUseCase,
		generateInviteUseCase, acceptInviteUseCase,
	)

	// Material platform adapters
	materialRepo := material.NewMaterialRepository(dbPool)
	groupProvider := material.NewGroupProvider(dbPool)
	groupMemberProvider := material.NewGroupMemberProvider(dbPool)
	authorProvider := material.NewAuthorProvider(dbPool)

	// Material use cases
	createMaterialUC := appMaterial.NewCreateMaterial(materialRepo, groupProvider, groupMemberProvider, authorProvider)
	updateMaterialUC := appMaterial.NewUpdateMaterial(materialRepo, groupProvider, authorProvider)
	getMaterialUC := appMaterial.NewGetMaterial(materialRepo, groupProvider, groupMemberProvider, authorProvider)
	listMaterialsUC := appMaterial.NewListMaterials(materialRepo, groupProvider, groupMemberProvider, authorProvider)
	publishMaterialUC := appMaterial.NewPublishMaterial(materialRepo, groupProvider, authorProvider)
	unpublishMaterialUC := appMaterial.NewUnpublishMaterial(materialRepo, groupProvider, authorProvider)
	pinMaterialUC := appMaterial.NewPinMaterial(materialRepo, groupProvider, groupMemberProvider, authorProvider)
	unpinMaterialUC := appMaterial.NewUnpinMaterial(materialRepo, groupProvider, groupMemberProvider, authorProvider)
	deleteMaterialUC := appMaterial.NewDeleteMaterial(materialRepo, groupProvider)

	materialHandler := handlerMaterial.NewHandler(
		createMaterialUC, updateMaterialUC, getMaterialUC, listMaterialsUC,
		publishMaterialUC, unpublishMaterialUC, pinMaterialUC, unpinMaterialUC,
		deleteMaterialUC,
	)

	router := adapterhttp.NewRouter(&adapterhttp.Handlers{
		Problem:  problemHandler,
		User:     userHandler,
		Auth:     authHandler,
		Group:    groupHandler,
		Material: materialHandler,
	}, &adapterhttp.Services{
		TokenService:       jwtService,
		SessionInvalidator: sessionInvalidator,
	}, cfg.AllowedOrigins)

	slog.Info("http starting", "port", cfg.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", cfg.Port), router); err != nil {
		slog.Error("http failed to start", "error", err)
		os.Exit(1)
	}
}
