package handler

import (
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/domain/ratelimit"
)

type UserHandler struct {
	createUser          *appuser.CreateUserUseCase
	getUserProfile      *appuser.GetUserProfileUseCase
	updateUser          *appuser.UpdateUserUseCase
	updatePassword      *appuser.UpdatePasswordUseCase
	adminUpdateUser     *appuser.AdminUpdateUserUseCase
	adminDeactivateUser *appuser.AdminDeactivateUserUseCase
	listUsers             *appuser.ListUsersUseCase
	requestEmailChange    *appuser.RequestEmailChangeUseCase
	confirmEmailChange    *appuser.ConfirmEmailChangeUseCase
	requestPasswordRecovery *appuser.RequestPasswordRecoveryUseCase
	resetPassword           *appuser.ResetPasswordUseCase
	requestDeactivation   *appuser.RequestDeactivationUseCase
	confirmDeactivation   *appuser.ConfirmDeactivationUseCase
	rateLimiter           ratelimit.RateLimiter
}

func NewUserHandler(
	createUser *appuser.CreateUserUseCase,
	getUserProfile *appuser.GetUserProfileUseCase,
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
	rateLimiter ratelimit.RateLimiter,
) *UserHandler {
	return &UserHandler{
		createUser:            createUser,
		getUserProfile:        getUserProfile,
		updateUser:            updateUser,
		updatePassword:        updatePassword,
		adminUpdateUser:       adminUpdateUser,
		adminDeactivateUser:   adminDeactivateUser,
		listUsers:             listUsers,
		requestEmailChange:    requestEmailChange,
		confirmEmailChange:    confirmEmailChange,
		requestPasswordRecovery: requestPasswordRecovery,
		resetPassword:           resetPassword,
		requestDeactivation:   requestDeactivation,
		confirmDeactivation:   confirmDeactivation,
		rateLimiter:           rateLimiter,
	}
}
