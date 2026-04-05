package user

import (
	"context"
	"errors"

	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type UpdateUserInput struct {
	UserID      string
	Name        *string
	Nickname    *string
	Institution *string
}

type UpdateUserUseCase struct {
	repo user.UserRepository
}

func NewUpdateUserUseCase(repo user.UserRepository) *UpdateUserUseCase {
	return &UpdateUserUseCase{repo: repo}
}

func (uc *UpdateUserUseCase) Execute(ctx context.Context, input UpdateUserInput) (*user.User, error) {
	if input.Name == nil && input.Nickname == nil && input.Institution == nil {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "body", Message: "At least one updatable field must be provided"},
		})
	}

	foundUser, err := uc.repo.FindByID(ctx, input.UserID)
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

	if len(fieldErrors) > 0 {
		return nil, apperror.NewValidation(fieldErrors)
	}

	foundUser.Update(nameToUpdate, nicknameToUpdate, institutionToUpdate, nil, nil)

	if err := uc.repo.Update(ctx, foundUser); err != nil {
		if errors.Is(err, user.ErrNicknameConflict) {
			return nil, apperror.NewConflict("NICKNAME_ALREADY_EXISTS", "The nickname is already in use")
		}
		return nil, apperror.NewInternal()
	}

	return foundUser, nil
}
