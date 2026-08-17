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
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	googleStorage "cloud.google.com/go/storage"
	"github.com/redis/go-redis/v9"
	"github.com/training-judge-center/backend/internal/adapter/auth"
	platformConfig "github.com/training-judge-center/backend/internal/adapter/config"
	adaptercontest "github.com/training-judge-center/backend/internal/adapter/contest"
	"github.com/training-judge-center/backend/internal/adapter/email"
	"github.com/training-judge-center/backend/internal/adapter/group"
	adapterhttp "github.com/training-judge-center/backend/internal/adapter/http"
	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	handlercontest "github.com/training-judge-center/backend/internal/adapter/http/handler/contest"
	handlerGroup "github.com/training-judge-center/backend/internal/adapter/http/handler/group"
	handlerMaterial "github.com/training-judge-center/backend/internal/adapter/http/handler/material"
	handlerProblem "github.com/training-judge-center/backend/internal/adapter/http/handler/problem"
	handlersubmission "github.com/training-judge-center/backend/internal/adapter/http/handler/submission"
	handlerteam "github.com/training-judge-center/backend/internal/adapter/http/handler/team"
	handlerUser "github.com/training-judge-center/backend/internal/adapter/http/handler/user"
	"github.com/training-judge-center/backend/internal/adapter/material"
	"github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/internal/adapter/problem"
	adapterqueue "github.com/training-judge-center/backend/internal/adapter/queue"
	"github.com/training-judge-center/backend/internal/adapter/ratelimit"
	adaptersubmission "github.com/training-judge-center/backend/internal/adapter/submission"
	adapterteam "github.com/training-judge-center/backend/internal/adapter/team"
	"github.com/training-judge-center/backend/internal/adapter/user"

	appcontest "github.com/training-judge-center/backend/internal/application/contest"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
	appteam "github.com/training-judge-center/backend/internal/application/team"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/config"
	"github.com/training-judge-center/backend/pkg/apperror"
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
	problemRepo := problem.NewRepository(dbPool)
	settingsProvider, err := platformConfig.NewPlatformSettings(cfg.VirtualObject)
	if err != nil {
		slog.Error("invalid platform settings in config", "error", err)
		os.Exit(1)
	}

	// File Storage
	var fileStorage appProblem.ProblemFileRepository
	var sourceCodeReadFn func(ctx context.Context, path string) ([]byte, error)
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
		fileStorage = problem.NewGCSFileRepository(gcsClient, cfg.GCSBucket)
		capturedClient, capturedBucket := gcsClient, cfg.GCSBucket
		sourceCodeReadFn = func(ctx context.Context, path string) ([]byte, error) {
			rc, err := capturedClient.Bucket(capturedBucket).Object(path).NewReader(ctx)
			if err != nil {
				slog.ErrorContext(ctx, "gcs: failed to open source code reader", "path", path, "error", err)
				return nil, apperror.NewInternal()
			}
			defer rc.Close()
			data, err := io.ReadAll(rc)
			if err != nil {
				slog.ErrorContext(ctx, "gcs: failed to read source code", "path", path, "error", err)
				return nil, apperror.NewInternal()
			}
			return data, nil
		}
	default:
		localDir := cfg.StorageLocalDir
		localRepo, err := problem.NewLocalFileRepository(localDir)
		if err != nil {
			slog.Error("failed to create local file storage", "error", err)
			os.Exit(1)
		}
		slog.Info("using local storage backend", "dir", localDir)
		fileStorage = localRepo
		capturedDir := localDir
		sourceCodeReadFn = func(ctx context.Context, path string) ([]byte, error) {
			fullPath := filepath.Join(capturedDir, path)
			data, err := os.ReadFile(fullPath)
			if err != nil {
				slog.ErrorContext(ctx, "local: failed to read source code", "path", fullPath, "error", err)
				return nil, apperror.NewInternal()
			}
			return data, nil
		}
	}

	icpcParser := problem.NewICPCParser(
		settingsProvider.MaxFileSizeTestCaseMB(),
		settingsProvider.MaxFileSizeDefaultMB(),
		settingsProvider.MaxFileCountTestCase(),
		settingsProvider.MaxFileCountSample(),
		cfg.VirtualObject.LanguageExtensions,
	)
	zipParser := problem.NewZipParser(icpcParser)
	packageParser := problem.NewICPCPackageParser(icpcParser)

	userProvider := problem.NewUserProvider(dbPool)
	problemStatisticsProvider := problem.NewStatisticsProvider(dbPool)
	problemActiveContestChecker := problem.NewActiveContestChecker(dbPool)

	// Problem use cases
	createProblemUseCase := appProblem.NewCreateProblemUseCase(problemRepo, settingsProvider)
	importProblemUseCase := appProblem.NewImportProblemUseCase(problemRepo, fileStorage, packageParser, settingsProvider)
	updateProblemUseCase := appProblem.NewUpdateProblemUseCase(problemRepo, settingsProvider)
	uploadProblemFilesUseCase := appProblem.NewUploadProblemFilesUseCase(problemRepo, fileStorage, zipParser, settingsProvider)
	deleteProblemFileUseCase := appProblem.NewDeleteProblemFileUseCase(problemRepo, fileStorage)
	addModifierUseCase := appProblem.NewAddModifierUseCase(problemRepo, userProvider)
	removeModifierUseCase := appProblem.NewRemoveModifierUseCase(problemRepo, userProvider)
	listModifiersUseCase := appProblem.NewListModifiersUseCase(problemRepo, userProvider)
	getProblemUseCase := appProblem.NewGetProblemUseCase(problemRepo, userProvider)
	listProblemsUseCase := appProblem.NewListProblemsUseCase(problemRepo, userProvider)
	unpublishProblemUseCase := appProblem.NewUnpublishProblemUseCase(problemRepo, problemActiveContestChecker)
	changeAccessibilityUseCase := appProblem.NewChangeAccessibilityUseCase(problemRepo)
	deleteProblemUseCase := appProblem.NewDeleteProblemUseCase(problemRepo, fileStorage, problemActiveContestChecker)
	getProblemStatisticsUseCase := appProblem.NewGetProblemStatisticsUseCase(problemRepo, problemStatisticsProvider)

	// User platform adapters
	userRepo := user.NewRepository(dbPool)
	refreshTokenRepo := user.NewRefreshTokenRepository(dbPool)
	passwordRecoveryRepo := user.NewPasswordRecoveryRepository(dbPool)
	emailChangeRepo := user.NewEmailChangeRepository(dbPool)
	deactivationRequestRepo := user.NewDeactivationRequestRepository(dbPool)
	deactivationAuditLogRepo := user.NewDeactivationAuditLogRepository(dbPool)
	oauthIdentityRepo := user.NewOAuthIdentityRepository(dbPool)

	// Infrastructure and cross-cutting services
	txManager := postgres.NewTransactionManager(dbPool)
	jwtService := auth.NewJWTService(cfg.JWTSecret, cfg.JWTExpirationHours)
	emailSender := email.NewSMTPSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom)
	redisRateLimiter := ratelimit.NewRedisRateLimiter(redisClient)
	sessionInvalidator := auth.NewRedisSessionInvalidator(redisClient, time.Duration(cfg.JWTExpirationHours)*time.Hour)
	rotationCache, err := auth.NewRedisRotationCache(redisClient, cfg.RotationCacheEncryptionKey)
	if err != nil {
		slog.Error("failed to initialize rotation cache", "error", err)
		os.Exit(1)
	}
	refreshTokenCodec := auth.NewJWTRefreshTokenCodec(cfg.JWTSecret)
	googleVerifier := auth.NewGoogleVerifier(cfg.GoogleClientID)

	// User use cases
	createUserUseCase := appuser.NewCreateUserUseCase(userRepo)
	loginUseCase := appuser.NewLoginUseCase(userRepo, refreshTokenRepo, jwtService, refreshTokenCodec, redisRateLimiter)
	loginWithGoogleUseCase := appuser.NewLoginWithGoogleUseCase(userRepo, oauthIdentityRepo, refreshTokenRepo, jwtService, refreshTokenCodec, googleVerifier, txManager)
	linkGoogleIdentityUseCase := appuser.NewLinkGoogleIdentityUseCase(userRepo, oauthIdentityRepo, googleVerifier, emailSender)
	unlinkGoogleIdentityUseCase := appuser.NewUnlinkGoogleIdentityUseCase(userRepo, oauthIdentityRepo, emailSender)
	refreshUseCase := appuser.NewRefreshUseCase(refreshTokenRepo, userRepo, jwtService, refreshTokenCodec, redisRateLimiter, rotationCache)
	logoutUseCase := appuser.NewLogoutUseCase(refreshTokenRepo, sessionInvalidator, refreshTokenCodec)
	getMyProfileUseCase := appuser.NewGetMyProfileUseCase(userRepo, oauthIdentityRepo)
	getUserByNicknameUseCase := appuser.NewGetUserByNicknameUseCase(userRepo)
	updateUserUseCase := appuser.NewUpdateUserUseCase(userRepo)
	updatePasswordUseCase := appuser.NewUpdatePasswordUseCase(userRepo, emailSender, sessionInvalidator, redisRateLimiter, refreshTokenRepo)
	setPasswordUseCase := appuser.NewSetPasswordUseCase(userRepo, emailSender)
	adminUpdateUserUseCase := appuser.NewAdminUpdateUserUseCase(userRepo)
	adminDeactivateUserUseCase := appuser.NewAdminDeactivateUserUseCase(userRepo, sessionInvalidator, refreshTokenRepo)
	listUsersUseCase := appuser.NewListUsersUseCase(userRepo)
	searchUsersUseCase := appuser.NewSearchUsersUseCase(userRepo)
	requestEmailChangeUseCase := appuser.NewRequestEmailChangeUseCase(userRepo, emailChangeRepo, emailSender, redisRateLimiter)
	confirmEmailChangeUseCase := appuser.NewConfirmEmailChangeUseCase(userRepo, emailChangeRepo, emailSender, txManager)
	requestPasswordRecoveryUseCase := appuser.NewRequestPasswordRecoveryUseCase(userRepo, passwordRecoveryRepo, emailSender, redisRateLimiter)
	resetPasswordUseCase := appuser.NewResetPasswordUseCase(userRepo, passwordRecoveryRepo, sessionInvalidator, txManager, refreshTokenRepo)
	requestDeactivationUseCase := appuser.NewRequestDeactivationUseCase(userRepo, deactivationRequestRepo, emailSender)
	confirmDeactivationUseCase := appuser.NewConfirmDeactivationUseCase(userRepo, deactivationRequestRepo, deactivationAuditLogRepo, emailSender, sessionInvalidator, txManager, refreshTokenRepo)

	// user dashboard adapters
	dashboardSubmissionProvider := adaptersubmission.NewDashboardProvider(dbPool)
	dashboardContestProvider := adaptercontest.NewDashboardProvider(dbPool)
	dashboardMaterialProvider := material.NewDashboardProvider(dbPool)
	problemsSolvedProvider := user.NewProblemsSolvedProvider(dbPool)

	// user dashboard use case
	getDashboardUseCase := appuser.NewGetDashboardUseCase(dashboardSubmissionProvider, dashboardContestProvider, dashboardMaterialProvider, problemsSolvedProvider)

	// user profile stats adapters
	rankingProvider := user.NewRankingProvider(dbPool)
	submissionStatsProvider := adaptersubmission.NewSubmissionStatsProvider(dbPool)
	contestParticipationProvider := adaptercontest.NewContestParticipationProvider(dbPool)
	topicStatsProvider := problem.NewTopicStatsProvider(dbPool)

	// user profile stats use case
	getProfileStatsUseCase := appuser.NewGetProfileStatsUseCase(rankingProvider, submissionStatsProvider, contestParticipationProvider, topicStatsProvider)

	// Handlers
	userHandler := handlerUser.NewHandler(createUserUseCase, getMyProfileUseCase, getUserByNicknameUseCase, updateUserUseCase, updatePasswordUseCase, adminUpdateUserUseCase, adminDeactivateUserUseCase, listUsersUseCase, requestEmailChangeUseCase, confirmEmailChangeUseCase, requestPasswordRecoveryUseCase, resetPasswordUseCase, requestDeactivationUseCase, confirmDeactivationUseCase, getDashboardUseCase, getProfileStatsUseCase, searchUsersUseCase, linkGoogleIdentityUseCase, unlinkGoogleIdentityUseCase, setPasswordUseCase)
	authHandler := handler.NewAuthHandler(loginUseCase, loginWithGoogleUseCase, refreshUseCase, logoutUseCase)

	// Group repositories & platform adapters
	groupRepo := group.NewRepository(dbPool)
	groupMemberRepo := group.NewMemberRepository(dbPool)
	groupUserProvider := group.NewUserProvider(dbPool)
	groupPrefsReader := group.NewPreferencesReader(dbPool)
	groupNicknameResolver := group.NewNicknameResolver(dbPool)
	groupEmailResolver := group.NewEmailResolver(dbPool)
	joinRequestRepo := group.NewJoinRequestRepository(dbPool)
	groupInvitationRepo := group.NewInvitationRepository(dbPool)
	groupDeletionProvider := group.NewDeletionProvider(dbPool)
	// Group use cases
	createGroupUseCase := appGroup.NewCreateGroupUseCase(groupRepo, groupMemberRepo, groupNicknameResolver, txManager)
	listGroupsUseCase := appGroup.NewListGroupsUseCase(groupRepo, groupMemberRepo)
	getGroupUseCase := appGroup.NewGetGroupUseCase(groupRepo, groupMemberRepo, groupUserProvider)
	listMyGroupsUseCase := appGroup.NewListMyGroupsUseCase(groupRepo, groupMemberRepo, groupPrefsReader)
	joinGroupUseCase := appGroup.NewJoinGroupUseCase(groupRepo, groupMemberRepo)
	requestJoinUseCase := appGroup.NewRequestJoinUseCase(groupRepo, groupMemberRepo, joinRequestRepo)
	approveRequestUseCase := appGroup.NewApproveRequestUseCase(groupMemberRepo, joinRequestRepo, txManager)
	rejectRequestUseCase := appGroup.NewRejectRequestUseCase(groupMemberRepo, joinRequestRepo)
	listJoinRequestsUseCase := appGroup.NewListJoinRequestsUseCase(groupMemberRepo, joinRequestRepo, groupUserProvider)
	getMyRequestUseCase := appGroup.NewGetMyRequestUseCase(joinRequestRepo)
	cancelMyRequestUseCase := appGroup.NewCancelMyRequestUseCase(joinRequestRepo)

	generateInviteUseCase := appGroup.NewGenerateInviteUseCase(groupRepo, groupMemberRepo, groupInvitationRepo, groupNicknameResolver, groupEmailResolver, groupUserProvider, txManager, emailSender, cfg.FrontendBaseURL)
	acceptInviteUseCase := appGroup.NewAcceptInviteUseCase(groupRepo, groupMemberRepo, groupInvitationRepo, txManager)
	listGroupInvitationsUseCase := appGroup.NewListGroupInvitationsUseCase(groupMemberRepo, groupInvitationRepo, groupUserProvider)
	revokeInvitationUseCase := appGroup.NewRevokeInvitationUseCase(groupMemberRepo, groupInvitationRepo)
	inviteByNicknamesUseCase := appGroup.NewInviteByNicknamesUseCase(groupRepo, groupMemberRepo, groupInvitationRepo, groupNicknameResolver, txManager, emailSender, cfg.FrontendBaseURL)

	addMemberUseCase := appGroup.NewAddMemberUseCase(groupRepo, groupMemberRepo, groupNicknameResolver)
	groupContestCleaner := group.NewContestRegistrationCleaner(dbPool)
	groupTeamSelectionCleaner := group.NewTeamSelectionCleaner(dbPool)
	removeMemberUseCase := appGroup.NewRemoveMemberUseCase(groupRepo, groupMemberRepo, groupNicknameResolver, groupContestCleaner, groupTeamSelectionCleaner, txManager)
	changeRoleUseCase := appGroup.NewChangeRoleUseCase(groupRepo, groupMemberRepo, groupNicknameResolver)
	leaveGroupUseCase := appGroup.NewLeaveGroupUseCase(groupRepo, groupMemberRepo, groupContestCleaner, groupTeamSelectionCleaner, txManager)
	listMembersUseCase := appGroup.NewListMembersUseCase(groupRepo, groupMemberRepo, groupUserProvider)
	groupStandingsCache := adaptercontest.NewStandingsCache(redisClient)
	deleteGroupUseCase := appGroup.NewDeleteGroupUseCase(groupRepo, groupMemberRepo, groupDeletionProvider, groupStandingsCache, txManager)
	updateGroupUseCase := appGroup.NewUpdateGroupUseCase(groupRepo, groupMemberRepo, joinRequestRepo, txManager)

	groupHandler := handlerGroup.NewHandler(
		createGroupUseCase, listGroupsUseCase, getGroupUseCase, listMyGroupsUseCase,
		joinGroupUseCase,
		requestJoinUseCase, approveRequestUseCase, rejectRequestUseCase,
		listJoinRequestsUseCase, getMyRequestUseCase, cancelMyRequestUseCase,
		generateInviteUseCase, acceptInviteUseCase, listGroupInvitationsUseCase, revokeInvitationUseCase, inviteByNicknamesUseCase,
		addMemberUseCase, removeMemberUseCase, changeRoleUseCase, leaveGroupUseCase, listMembersUseCase,
		deleteGroupUseCase,
		updateGroupUseCase,
	)

	// Material platform adapters
	materialRepo := material.NewRepository(dbPool)
	groupProvider := material.NewGroupProvider(dbPool)
	groupMemberProvider := material.NewGroupMemberProvider(dbPool)
	authorProvider := material.NewAuthorProvider(dbPool)
	authorIDProvider := material.NewAuthorIDProvider(dbPool)

	// Material use cases
	createMaterialUseCase := appMaterial.NewCreateMaterialUseCase(materialRepo, groupProvider, groupMemberProvider, authorProvider)
	updateMaterialUseCase := appMaterial.NewUpdateMaterialUseCase(materialRepo, groupProvider, authorProvider)
	getMaterialUseCase := appMaterial.NewGetMaterialUseCase(materialRepo, groupProvider, groupMemberProvider, authorProvider)
	listMaterialsUseCase := appMaterial.NewListMaterialsUseCase(materialRepo, groupProvider, groupMemberProvider, authorProvider, authorIDProvider)
	publishMaterialUseCase := appMaterial.NewPublishMaterialUseCase(materialRepo, groupProvider, authorProvider)
	unpublishMaterialUseCase := appMaterial.NewUnpublishMaterialUseCase(materialRepo, groupProvider, authorProvider)
	pinMaterialUseCase := appMaterial.NewPinMaterialUseCase(materialRepo, groupProvider, groupMemberProvider, authorProvider)
	unpinMaterialUseCase := appMaterial.NewUnpinMaterialUseCase(materialRepo, groupProvider, groupMemberProvider, authorProvider)
	deleteMaterialUseCase := appMaterial.NewDeleteMaterialUseCase(materialRepo, groupProvider)

	materialHandler := handlerMaterial.NewHandler(
		createMaterialUseCase, updateMaterialUseCase, getMaterialUseCase, listMaterialsUseCase,
		publishMaterialUseCase, unpublishMaterialUseCase, pinMaterialUseCase, unpinMaterialUseCase,
		deleteMaterialUseCase,
	)

	// contest adapters
	contestRepo := adaptercontest.NewRepository(dbPool)
	contestGroupProvider := adaptercontest.NewGroupProvider(dbPool)
	contestMemberProvider := adaptercontest.NewGroupMemberProvider(dbPool)
	contestProblemProvider := adaptercontest.NewProblemProvider(dbPool)
	contestOwnerProvider := adaptercontest.NewOwnerProvider(dbPool)
	contestProfileProvider := adaptercontest.NewParticipantProfileProvider(dbPool)
	contestRegistrationRepo := adaptercontest.NewRegistrationRepository(dbPool)
	contestParticipantProvider := adaptercontest.NewContestParticipantProvider(contestRegistrationRepo)
	contestNicknameProvider := adaptercontest.NewParticipantNicknameProvider(dbPool)
	contestStandingsCache := adaptercontest.NewStandingsCache(redisClient)
	contestSubmissionProvider := adaptercontest.NewStandingsSubmissionProvider(dbPool)
	contestSubmissionsProvider := adaptercontest.NewContestSubmissionProvider(dbPool)
	contestTeamParticipationRepo := adaptercontest.NewTeamParticipationRepository(dbPool)
	contestCallerStandingProvider := adaptercontest.NewCallerStandingProvider(dbPool)

	// contest use cases
	createContestUseCase := appcontest.NewCreateContestUseCase(
		contestRepo, contestGroupProvider, contestMemberProvider,
		contestProblemProvider, contestOwnerProvider,
	)
	updateContestUseCase := appcontest.NewUpdateContestUseCase(
		contestRepo, contestGroupProvider, contestMemberProvider,
		contestProblemProvider, contestOwnerProvider, contestTeamParticipationRepo, txManager,
	)
	getContestUseCase := appcontest.NewGetContestUseCase(
		contestRepo, contestGroupProvider, contestMemberProvider,
		contestProblemProvider, contestOwnerProvider, contestParticipantProvider,
	)
	listContestsUseCase := appcontest.NewListContestsUseCase(
		contestRepo, contestGroupProvider, contestMemberProvider, contestParticipantProvider,
	)
	listMyContestsUseCase := appcontest.NewListMyContestsUseCase(
		contestRepo, contestGroupProvider, contestParticipantProvider,
	)
	contestTeamSelectionChecker := adaptercontest.NewTeamSelectionChecker(dbPool)
	registerToContestUseCase := appcontest.NewRegisterToContestUseCase(
		contestRepo, contestRegistrationRepo, contestMemberProvider, contestTeamSelectionChecker,
	)
	unregisterFromContestUseCase := appcontest.NewUnregisterFromContestUseCase(
		contestRepo, contestRegistrationRepo, contestMemberProvider,
	)
	getRegistrationStatusUseCase := appcontest.NewGetRegistrationStatusUseCase(
		contestRepo, contestRegistrationRepo,
	)
	listContestRegistrationsUseCase := appcontest.NewListContestRegistrationsUseCase(
		contestRepo, contestRegistrationRepo, contestMemberProvider, contestNicknameProvider,
	)
	getStandingsUseCase := appcontest.NewGetStandingsUseCase(
		contestRepo, contestRegistrationRepo, contestSubmissionProvider,
		contestTeamParticipationRepo, contestProfileProvider,
		contestGroupProvider, contestMemberProvider,
		contestStandingsCache,
		30*time.Second,
	)
	listContestSubmissionsUseCase := appcontest.NewListContestSubmissionsUseCase(
		contestRepo, contestGroupProvider, contestMemberProvider,
		contestParticipantProvider, contestSubmissionsProvider,
		contestCallerStandingProvider,
	)
	deleteContestUseCase := appcontest.NewDeleteContestUseCase(
		contestRepo, contestMemberProvider, contestStandingsCache,
	)
	contestHandler := handlercontest.NewHandler(
		createContestUseCase, updateContestUseCase, deleteContestUseCase,
		getContestUseCase, listContestsUseCase, listMyContestsUseCase,
		registerToContestUseCase, unregisterFromContestUseCase,
		getRegistrationStatusUseCase, listContestRegistrationsUseCase,
		getStandingsUseCase, listContestSubmissionsUseCase,
	)

	// team adapters
	teamRepo := adapterteam.NewRepository(dbPool)
	teamMemberRepo := adapterteam.NewMemberRepository(dbPool)
	teamInvitationRepo := adapterteam.NewInvitationRepository(dbPool)
	teamUserProvider := adapterteam.NewUserProvider(dbPool)
	teamContestParticipationChecker := adapterteam.NewContestParticipationChecker(dbPool)
	teamContestProvider := adapterteam.NewContestProvider(dbPool)
	teamIndivChecker := adapterteam.NewIndividualRegistrationChecker(dbPool)
	teamGroupMemberChecker := adapterteam.NewGroupMemberChecker(dbPool)
	// team use cases
	createTeamUseCase := appteam.NewCreateTeamUseCase(teamRepo, teamMemberRepo, teamUserProvider, txManager)
	listMyTeamsUseCase := appteam.NewListMyTeamsUseCase(teamRepo, teamMemberRepo)
	getTeamUseCase := appteam.NewGetTeamUseCase(teamRepo, teamMemberRepo, teamInvitationRepo, teamUserProvider)
	inviteToTeamUseCase := appteam.NewInviteToTeamUseCase(teamRepo, teamMemberRepo, teamInvitationRepo, teamUserProvider)
	listMyInvitationsUseCase := appteam.NewListMyInvitationsUseCase(teamInvitationRepo, teamRepo, teamUserProvider)
	acceptInvitationUseCase := appteam.NewAcceptInvitationUseCase(teamInvitationRepo, teamMemberRepo, txManager)
	rejectInvitationUseCase := appteam.NewRejectInvitationUseCase(teamInvitationRepo)
	teamScheduledCleaner := adapterteam.NewScheduledParticipationCleaner(dbPool)
	leaveTeamUseCase := appteam.NewLeaveTeamUseCase(teamMemberRepo, teamContestParticipationChecker, teamScheduledCleaner, txManager)
	registerTeamToContestUseCase := appteam.NewRegisterTeamToContestUseCase(
		teamMemberRepo, teamRepo, teamContestProvider, contestTeamParticipationRepo,
		teamIndivChecker, teamGroupMemberChecker, teamUserProvider, txManager,
	)
	updateTeamRegistrationUseCase := appteam.NewUpdateTeamRegistrationUseCase(
		teamMemberRepo, teamRepo, teamContestProvider, contestTeamParticipationRepo,
		teamIndivChecker, teamGroupMemberChecker, teamUserProvider, txManager,
	)
	unregisterTeamFromContestUseCase := appteam.NewUnregisterTeamFromContestUseCase(
		teamMemberRepo, teamContestProvider, contestTeamParticipationRepo,
	)
	listTeamRegistrationsUseCase := appteam.NewListTeamRegistrationsUseCase(
		teamContestProvider, teamGroupMemberChecker, contestTeamParticipationRepo, teamRepo, teamUserProvider,
	)

	teamHandler := handlerteam.NewHandler(
		createTeamUseCase,
		listMyTeamsUseCase,
		getTeamUseCase,
		inviteToTeamUseCase,
		listMyInvitationsUseCase,
		acceptInvitationUseCase,
		rejectInvitationUseCase,
		leaveTeamUseCase,
		registerTeamToContestUseCase,
		updateTeamRegistrationUseCase,
		unregisterTeamFromContestUseCase,
		listTeamRegistrationsUseCase,
	)

	// submission adapters
	submissionRepo := adaptersubmission.NewRepository(dbPool)
	submissionProblemProvider := adaptersubmission.NewProblemProvider(dbPool)
	submissionStorage := adaptersubmission.NewSourceStorage(fileStorage.UploadFile, fileStorage.DeleteFile)
	submissionSourceReader := adaptersubmission.NewSourceCodeReader(sourceCodeReadFn)
	submissionProblemDisplay := adaptersubmission.NewProblemDisplayProvider(dbPool)
	submissionUserDisplay := adaptersubmission.NewUserProvider(dbPool)
	submissionContestDisplay := adaptersubmission.NewContestProvider(dbPool)
	submissionLeadChecker := adaptersubmission.NewLeadChecker(dbPool)
	submissionTeamChecker := adaptersubmission.NewTeamMembershipChecker(dbPool)
	submissionContestSubmissionProvider := adaptersubmission.NewContestSubmissionProvider(dbPool)
	submissionStandingIDResolver := adaptersubmission.NewStandingIDResolver(dbPool)

	// submission queue — RabbitMQ when URL is set, no-op otherwise
	var submissionQueue appsubmission.SubmissionQueue
	if cfg.RabbitMQURL != "" {
		rmq, err := adapterqueue.NewRabbitMQQueue(cfg.RabbitMQURL)
		if err != nil {
			slog.Error("failed to connect to RabbitMQ", "error", err)
			os.Exit(1)
		}
		defer rmq.Close()
		submissionQueue = rmq
		slog.Info("using RabbitMQ submission queue")
	} else {
		submissionQueue = adaptersubmission.NoOpQueue{}
		slog.Info("using no-op submission queue (RABBITMQ_URL not set)")
	}

	submissionRejudger := adaptersubmission.NewRejudger(dbPool, submissionQueue)
	contestRejudgeProvider := adaptercontest.NewContestRejudgeProvider(dbPool)
	rejudgeSubmissionsUseCase := appProblem.NewRejudgeSubmissionsUseCase(problemRepo, submissionRejudger)
	rejudgeContestSubmissionsUseCase := appProblem.NewRejudgeContestSubmissionsUseCase(problemRepo, submissionRejudger, contestRejudgeProvider)
	adminRejudgeSubmissionsUseCase := appProblem.NewAdminRejudgeSubmissionsUseCase(problemRepo, submissionRejudger, contestRejudgeProvider)

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
		getProblemStatisticsUseCase,
		rejudgeSubmissionsUseCase,
		rejudgeContestSubmissionsUseCase,
		adminRejudgeSubmissionsUseCase,
		userProvider,
		settingsProvider,
	)

	// submission use cases
	submitSolutionUseCase := appsubmission.NewSubmitSolutionUseCase(
		submissionProblemProvider,
		submissionRepo,
		submissionStorage,
		submissionQueue,
		1<<20, // 1 MB
		1,     // 1 second rate limit
	)
	submitContestSolutionUseCase := appsubmission.NewSubmitContestSolutionUseCase(
		submissionContestSubmissionProvider,
		submissionStandingIDResolver,
		submissionProblemProvider,
		submissionRepo,
		submissionStorage,
		submissionQueue,
		1<<20, // 1 MB
		1,     // 1 second rate limit
	)
	getSubmissionUseCase := appsubmission.NewGetSubmissionUseCase(
		submissionRepo,
		submissionSourceReader,
		submissionProblemDisplay,
		submissionUserDisplay,
		submissionContestDisplay,
		submissionTeamChecker,
		submissionLeadChecker,
	)
	updateSubmissionVisibilityUseCase := appsubmission.NewUpdateSubmissionVisibilityUseCase(submissionRepo)
	listMySubmissionsUseCase := appsubmission.NewListMySubmissionsUseCase(
		submissionRepo,
		submissionProblemDisplay,
		submissionUserDisplay,
		submissionContestDisplay,
		submissionProblemProvider,
	)
	listProblemSubmissionsUseCase := appsubmission.NewListProblemSubmissionsUseCase(
		submissionRepo,
		submissionProblemProvider,
		submissionUserDisplay,
	)

	submissionProblemJudgingProvider := adaptersubmission.NewProblemJudgingProvider(dbPool)
	submissionContestTimesProvider := adaptersubmission.NewContestTimesProvider(dbPool)
	rejudgeSubmissionUseCase := appsubmission.NewRejudgeSubmissionUseCase(
		submissionRepo,
		submissionProblemJudgingProvider,
		submissionContestTimesProvider,
		submissionRejudger,
	)

	submissionHandler := handlersubmission.NewHandler(
		submitSolutionUseCase,
		submitContestSolutionUseCase,
		getSubmissionUseCase,
		updateSubmissionVisibilityUseCase,
		listMySubmissionsUseCase,
		listProblemSubmissionsUseCase,
		rejudgeSubmissionUseCase,
	)

	router := adapterhttp.NewRouter(&adapterhttp.Handlers{
		Problem:    problemHandler,
		User:       userHandler,
		Auth:       authHandler,
		Group:      groupHandler,
		Material:   materialHandler,
		Contest:    contestHandler,
		Team:       teamHandler,
		Submission: submissionHandler,
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
