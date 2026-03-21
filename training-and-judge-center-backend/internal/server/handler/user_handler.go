package handler

import (
	appuser "github.com/training-judge-center/backend/internal/application/user"
)

type UserHandler struct {
	createUser          *appuser.CreateUserUseCase
	getUserProfile      *appuser.GetUserProfileUseCase
	updateUser          *appuser.UpdateUserUseCase
	updatePassword      *appuser.UpdatePasswordUseCase
	adminUpdateUser     *appuser.AdminUpdateUserUseCase
	adminDeactivateUser *appuser.AdminDeactivateUserUseCase
}

func NewUserHandler(
	createUser *appuser.CreateUserUseCase,
	getUserProfile *appuser.GetUserProfileUseCase,
	updateUser *appuser.UpdateUserUseCase,
	updatePassword *appuser.UpdatePasswordUseCase,
	adminUpdateUser *appuser.AdminUpdateUserUseCase,
	adminDeactivateUser *appuser.AdminDeactivateUserUseCase,
) *UserHandler {
	return &UserHandler{
		createUser:          createUser,
		getUserProfile:      getUserProfile,
		updateUser:          updateUser,
		updatePassword:      updatePassword,
		adminUpdateUser:     adminUpdateUser,
		adminDeactivateUser: adminDeactivateUser,
	}
}
