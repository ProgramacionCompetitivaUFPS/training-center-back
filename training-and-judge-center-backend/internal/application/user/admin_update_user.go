package user

import (
	"context"
	"errors"

	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type AdminUpdateUserInput struct {
	TargetID    string
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

func (uc *AdminUpdateUserUseCase) Execute(ctx context.Context, input AdminUpdateUserInput) (*user.User, error) {
	if input.Name == nil && input.Nickname == nil && input.Institution == nil && input.Email == nil && input.Role == nil {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "body", Message: "At least one updatable field must be provided"},
		})
	}

	foundUser, err := uc.repo.FindByID(ctx, input.TargetID)
	if err != nil {
		return nil, apperror.NewInternal()
	}
	if foundUser == nil {
		return nil, apperror.NewNotFound("NOT_FOUND", "User not found")
	}

	var fieldErrors []apperror.FieldError
	var nameToUpdate *string
	var nicknameToUpdate *user.Nickname
	var institutionToUpdate *string
	var emailToUpdate *user.Email
	var roleToUpdate *user.Role

	if input.Name != nil {
		if *input.Name == "" {
			fieldErrors = append(fieldErrors, apperror.FieldError{Field: "name", Message: "Name cannot be empty"})
		} else {
			nameToUpdate = input.Name
		}
	}

	if input.Nickname != nil {
		newNickname, err := user.NewNickname(*input.Nickname)
		if err != nil {
			fieldErrors = append(fieldErrors, apperror.FieldError{Field: "nickname", Message: err.Error()})
		} else if newNickname.String() != foundUser.Nickname().String() {
			nicknameToUpdate = &newNickname
		}
	}

	if input.Institution != nil {
		if *input.Institution == "" {
			fieldErrors = append(fieldErrors, apperror.FieldError{Field: "institution", Message: "Institution cannot be empty"})
		} else {
			institutionToUpdate = input.Institution
		}
	}

	if input.Email != nil {
		newEmail, err := user.NewEmail(*input.Email)
		if err != nil {
			fieldErrors = append(fieldErrors, apperror.FieldError{Field: "email", Message: err.Error()})
		} else if foundUser.Email() == nil || newEmail.String() != foundUser.Email().String() {
			emailToUpdate = &newEmail
		}
	}

	if input.Role != nil {
		newRole, err := user.NewRole(*input.Role)
		if err != nil {
			fieldErrors = append(fieldErrors, apperror.FieldError{Field: "role", Message: "Invalid role value"})
		} else if newRole == user.RoleAdmin {
			fieldErrors = append(fieldErrors, apperror.FieldError{Field: "role", Message: "Cannot assign ADMIN role through this endpoint"})
		} else {
			roleToUpdate = &newRole
		}
	}

	if len(fieldErrors) > 0 {
		return nil, apperror.NewValidation(fieldErrors)
	}

	if err := foundUser.Update(nameToUpdate, nicknameToUpdate, institutionToUpdate, emailToUpdate, roleToUpdate); err != nil {
		return nil, apperror.NewInternal()
	}

	if err := uc.repo.Update(ctx, foundUser); err != nil {
		if errors.Is(err, user.ErrNicknameConflict) {
			return nil, apperror.NewConflict("NICKNAME_ALREADY_EXISTS", "The nickname is already in use")
		}
		if errors.Is(err, user.ErrEmailConflict) {
			return nil, apperror.NewConflict("EMAIL_ALREADY_EXISTS", "The email is already in use")
		}
		return nil, apperror.NewInternal()
	}

	return foundUser, nil
}
