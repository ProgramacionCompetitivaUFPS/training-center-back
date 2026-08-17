package user

import (
	appuser "github.com/training-judge-center/backend/internal/application/user"
)

type Handler struct {
	createUser              *appuser.CreateUserUseCase
	getMyProfile            *appuser.GetMyProfileUseCase
	getUserByNickname       *appuser.GetUserByNicknameUseCase
	updateUser              *appuser.UpdateUserUseCase
	updatePassword          *appuser.UpdatePasswordUseCase
	adminUpdateUser         *appuser.AdminUpdateUserUseCase
	adminDeactivateUser     *appuser.AdminDeactivateUserUseCase
	listUsers               *appuser.ListUsersUseCase
	requestEmailChange      *appuser.RequestEmailChangeUseCase
	confirmEmailChange      *appuser.ConfirmEmailChangeUseCase
	requestPasswordRecovery *appuser.RequestPasswordRecoveryUseCase
	resetPassword           *appuser.ResetPasswordUseCase
	requestDeactivation     *appuser.RequestDeactivationUseCase
	confirmDeactivation     *appuser.ConfirmDeactivationUseCase
	getDashboard            *appuser.GetDashboardUseCase
	getProfileStats         *appuser.GetProfileStatsUseCase
	searchUsers             *appuser.SearchUsersUseCase
	linkGoogle              *appuser.LinkGoogleIdentityUseCase
	unlinkGoogle            *appuser.UnlinkGoogleIdentityUseCase
	setPassword             *appuser.SetPasswordUseCase
}

func NewHandler(
	createUser *appuser.CreateUserUseCase,
	getMyProfile *appuser.GetMyProfileUseCase,
	getUserByNickname *appuser.GetUserByNicknameUseCase,
	updateUser *appuser.UpdateUserUseCase,
	updatePassword *appuser.UpdatePasswordUseCase,
	adminUpdateUser *appuser.AdminUpdateUserUseCase,
	adminDeactivateUser *appuser.AdminDeactivateUserUseCase,
	listUsers *appuser.ListUsersUseCase,
	requestEmailChange *appuser.RequestEmailChangeUseCase,
	confirmEmailChange *appuser.ConfirmEmailChangeUseCase,
	requestPasswordRecovery *appuser.RequestPasswordRecoveryUseCase,
	resetPassword *appuser.ResetPasswordUseCase,
	requestDeactivation *appuser.RequestDeactivationUseCase,
	confirmDeactivation *appuser.ConfirmDeactivationUseCase,
	getDashboard *appuser.GetDashboardUseCase,
	getProfileStats *appuser.GetProfileStatsUseCase,
	searchUsers *appuser.SearchUsersUseCase,
	linkGoogle *appuser.LinkGoogleIdentityUseCase,
	unlinkGoogle *appuser.UnlinkGoogleIdentityUseCase,
	setPassword *appuser.SetPasswordUseCase,
) *Handler {
	return &Handler{
		createUser:              createUser,
		getMyProfile:            getMyProfile,
		getUserByNickname:       getUserByNickname,
		updateUser:              updateUser,
		updatePassword:          updatePassword,
		adminUpdateUser:         adminUpdateUser,
		adminDeactivateUser:     adminDeactivateUser,
		listUsers:               listUsers,
		requestEmailChange:      requestEmailChange,
		confirmEmailChange:      confirmEmailChange,
		requestPasswordRecovery: requestPasswordRecovery,
		resetPassword:           resetPassword,
		requestDeactivation:     requestDeactivation,
		confirmDeactivation:     confirmDeactivation,
		getDashboard:            getDashboard,
		getProfileStats:         getProfileStats,
		searchUsers:             searchUsers,
		linkGoogle:              linkGoogle,
		unlinkGoogle:            unlinkGoogle,
		setPassword:             setPassword,
	}
}
