package user

import (
	"context"
	"log/slog"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type UserProfileOutput struct {
	User          UserDTO
	IsFullProfile bool
}

type GetUserProfileUseCase struct {
	repo user.Repository
}

func NewGetUserProfileUseCase(repo user.Repository) *GetUserProfileUseCase {
	return &GetUserProfileUseCase{repo: repo}
}

func (uc *GetUserProfileUseCase) GetMyProfile(ctx context.Context, userID string) (*UserProfileOutput, error) {
	foundUser, err := uc.repo.FindByID(ctx, userID)
	if err != nil {
		slog.Error("failed to find user profile by id", "user_id", userID, "error", err)
		return nil, apperror.NewInternal()
	}
	if foundUser == nil {
		return nil, apperror.NewNotFound("NOT_FOUND", "User not found")
	}

	return &UserProfileOutput{User: userToDTO(foundUser), IsFullProfile: true}, nil
}

func (uc *GetUserProfileUseCase) GetUserByNickname(ctx context.Context, requesterID string, requesterRole shared.Role, nickname string) (*UserProfileOutput, error) {
	parsedNickname, err := user.NewNickname(nickname)
	if err != nil {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "nickname", Message: err.Error()},
		})
	}

	targetUser, err := uc.repo.FindByNickname(ctx, parsedNickname)
	if err != nil {
		slog.Error("failed to find user profile by nickname", "nickname", nickname, "error", err)
		return nil, apperror.NewInternal()
	}
	if targetUser == nil {
		return nil, apperror.NewNotFound("NOT_FOUND", "User not found")
	}

	if targetUser.Status() == user.StatusDeactivated {
		return nil, apperror.NewNotFound("NOT_FOUND", "User not found")
	}

	isSelf := targetUser.ID() == requesterID
	if isSelf {
		return &UserProfileOutput{User: userToDTO(targetUser), IsFullProfile: true}, nil
	}

	isRequesterAdmin := requesterRole == shared.RoleAdmin

	if !isRequesterAdmin && targetUser.Role() == shared.RoleAdmin {
		return nil, apperror.NewForbidden("ADMIN_PROFILE_RESTRICTED", "Admin profiles are not accessible to non-admin users")
	}

	return &UserProfileOutput{User: userToDTO(targetUser), IsFullProfile: isRequesterAdmin}, nil
}
