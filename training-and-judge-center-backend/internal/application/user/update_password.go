package user

import (
	"context"
	"time"

	"github.com/training-judge-center/backend/internal/domain/notification"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type UpdatePasswordInput struct {
	CurrentPassword string
	NewPassword     string
}

type UpdatePasswordUseCase struct {
	repo        user.UserRepository
	emailSender notification.EmailSender
}

func NewUpdatePasswordUseCase(repo user.UserRepository, email notification.EmailSender) *UpdatePasswordUseCase {
	return &UpdatePasswordUseCase{repo: repo, emailSender: email}
}

func (uc *UpdatePasswordUseCase) Execute(ctx context.Context, userID string, input UpdatePasswordInput) error {
	foundUser, err := uc.repo.FindByID(ctx, userID)
	if err != nil {
		return apperror.NewInternal()
	}
	if foundUser == nil {
		return apperror.NewNotFound("NOT_FOUND", "User not found")
	}

	if !foundUser.Password.Compare(input.CurrentPassword) {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "currentPassword", Message: "Current password is incorrect"},
		})
	}

	newPassword, err := user.NewPassword(input.NewPassword)
	if err != nil {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "newPassword", Message: err.Error()},
		})
	}

	if foundUser.Password.Compare(input.NewPassword) {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "newPassword", Message: "New password must be different from current password"},
		})
	}

	now := time.Now()
	foundUser.Password = newPassword
	foundUser.UpdatedAt = &now

	if err := uc.repo.Update(ctx, foundUser); err != nil {
		return apperror.NewInternal()
	}

	subject := "Security Alert: Password Changed"
	body := "Your password has been changed successfully. If you did not make this change, please contact support immediately."
	_ = uc.emailSender.Send(ctx, foundUser.Email.String(), subject, body)

	return nil
}
