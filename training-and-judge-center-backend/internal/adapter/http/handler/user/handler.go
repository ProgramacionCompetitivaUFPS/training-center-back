package user

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
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
	}
}

func (h *Handler) requireCurrentUser(w http.ResponseWriter, r *http.Request) (*appshared.CurrentUser, bool) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteError(r.Context(), w, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, "Invalid or missing authentication token"))
		return nil, false
	}
	return &cu, true
}
