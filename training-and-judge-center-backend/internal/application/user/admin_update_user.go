package user

import (
	"context"
	"time"

	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type AdminUpdateUserInput struct {
	Name        *string
	Nickname    *string
	Institution *string
	Email       *string
	Role        *string
}

type AdminUpdateUserUseCase struct {
	repo user.UserRepository
}

func NewAdminUpdateUserUseCase(repo user.UserRepository) *AdminUpdateUserUseCase {
	return &AdminUpdateUserUseCase{repo: repo}
}

func (uc *AdminUpdateUserUseCase) Execute(ctx context.Context, targetID string, input AdminUpdateUserInput) (*user.User, error) {
	if input.Name == nil && input.Nickname == nil && input.Institution == nil && input.Email == nil && input.Role == nil {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "body", Message: "At least one updatable field must be provided"},
		})
	}

	foundUser, err := uc.repo.FindByID(ctx, targetID)
	if err != nil {
		return nil, apperror.NewInternal()
	}
	if foundUser == nil {
		return nil, apperror.NewNotFound("NOT_FOUND", "User not found")
	}

	var fieldErrors []apperror.FieldError

	if input.Name != nil {
		if *input.Name == "" {
			fieldErrors = append(fieldErrors, apperror.FieldError{Field: "name", Message: "Name cannot be empty"})
		} else {
			foundUser.Name = *input.Name
		}
	}

	if input.Nickname != nil {
		newNickname, err := user.NewNickname(*input.Nickname)
		if err != nil {
			fieldErrors = append(fieldErrors, apperror.FieldError{Field: "nickname", Message: err.Error()})
		} else if newNickname.String() != foundUser.Nickname.String() {
			exists, err := uc.repo.ExistsByNickname(ctx, newNickname)
			if err != nil {
				return nil, apperror.NewInternal()
			}
			if exists {
				fieldErrors = append(fieldErrors, apperror.FieldError{Field: "nickname", Message: "The nickname is already in use"})
			} else {
				foundUser.Nickname = newNickname
			}
		}
	}

	if input.Institution != nil {
		if *input.Institution == "" {
			fieldErrors = append(fieldErrors, apperror.FieldError{Field: "institution", Message: "Institution cannot be empty"})
		} else {
			foundUser.Institution = *input.Institution
		}
	}

	if input.Email != nil {
		newEmail, err := user.NewEmail(*input.Email)
		if err != nil {
			fieldErrors = append(fieldErrors, apperror.FieldError{Field: "email", Message: err.Error()})
		} else if foundUser.Email == nil || newEmail.String() != foundUser.Email.String() {
			exists, err := uc.repo.ExistsByEmail(ctx, newEmail)
			if err != nil {
				return nil, apperror.NewInternal()
			}
			if exists {
				fieldErrors = append(fieldErrors, apperror.FieldError{Field: "email", Message: "Email already exists"})
			} else {
				foundUser.Email = &newEmail
			}
		}
	}

	if input.Role != nil {
		newRole, err := user.NewRole(*input.Role)
		if err != nil {
			fieldErrors = append(fieldErrors, apperror.FieldError{Field: "role", Message: "Invalid role value"})
		} else if newRole == user.RoleAdmin {
			fieldErrors = append(fieldErrors, apperror.FieldError{Field: "role", Message: "Cannot assign ADMIN role through this endpoint"})
		} else {
			foundUser.Role = newRole
		}
	}

	if len(fieldErrors) > 0 {
		return nil, apperror.NewValidation(fieldErrors)
	}

	now := time.Now()
	foundUser.UpdatedAt = &now

	if err := uc.repo.Update(ctx, foundUser); err != nil {
		return nil, apperror.NewInternal()
	}

	return foundUser, nil
}
