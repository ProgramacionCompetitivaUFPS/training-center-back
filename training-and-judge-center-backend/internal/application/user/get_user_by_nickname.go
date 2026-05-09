package user

import (
	"context"
	"log/slog"

	"github.com/training-judge-center/backend/internal/domain/shared"
	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GetUserByNicknameInput struct {
	RequesterID   string
	RequesterRole shared.Role
	Nickname      string
}

type GetUserByNicknameUseCase struct {
	repo domain.Repository
}

func NewGetUserByNicknameUseCase(repo domain.Repository) *GetUserByNicknameUseCase {
	return &GetUserByNicknameUseCase{repo: repo}
}

func (uc *GetUserByNicknameUseCase) Execute(ctx context.Context, in GetUserByNicknameInput) (*UserProfileOutput, error) {
	parsedNickname, err := domain.NewNickname(in.Nickname)
	if err != nil {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "nickname", Message: err.Error()},
		})
	}

	targetUser, err := uc.repo.FindByNickname(ctx, parsedNickname)
	if err != nil {
		slog.ErrorContext(ctx, "failed to find user profile by nickname", "nickname", in.Nickname, "error", err)
		return nil, apperror.NewInternal()
	}
	
	if targetUser == nil || targetUser.Status() == domain.StatusDeactivated {
		return nil, apperror.NewNotFound("NOT_FOUND", "User not found")
	}

	isSelf := targetUser.ID() == in.RequesterID
	if isSelf {
		return &UserProfileOutput{User: userToDTO(targetUser), IsFullProfile: true}, nil
	}

	isRequesterAdmin := in.RequesterRole == shared.RoleAdmin

	if !isRequesterAdmin && targetUser.Role() == shared.RoleAdmin {
		return nil, apperror.NewForbidden("ADMIN_PROFILE_RESTRICTED", "Admin profiles are not accessible to non-admin users")
	}

	return &UserProfileOutput{User: userToDTO(targetUser), IsFullProfile: isRequesterAdmin}, nil
}
