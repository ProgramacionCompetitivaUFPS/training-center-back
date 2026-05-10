package user

import (
	"context"
	"log/slog"
	"time"

	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type UpdateUserInput struct {
	UserID      string
	Name        *string
	Nickname    *string
	Institution *string
	City        *string
	Country     *string
}

type UpdateUserUseCase struct {
	repo user.Repository
}

func NewUpdateUserUseCase(repo user.Repository) *UpdateUserUseCase {
	return &UpdateUserUseCase{repo: repo}
}

type UpdateUserOutput struct {
	User UserDTO
}

func (uc *UpdateUserUseCase) Execute(ctx context.Context, input UpdateUserInput) (*UpdateUserOutput, error) {
	if input.Name == nil && input.Nickname == nil && input.Institution == nil && input.City == nil && input.Country == nil {
		return nil,apperror.NewValidation([]apperror.FieldError{
			{Field: "body", Message: "At least one updatable field must be provided"},
		})
	}

	foundUser, err := uc.repo.FindByID(ctx, input.UserID)
	if err != nil {
		slog.Error("failed to find user by id during update", "user_id", input.UserID, "error", err)
		return nil,apperror.NewInternal()
	}
	if foundUser == nil {
		return nil,apperror.NewNotFound(user.ErrCodeUserNotFound, "User not found")
	}

	var fieldErrors []apperror.FieldError
	var nameToUpdate *string
	var nicknameToUpdate *user.Nickname
	var institutionToUpdate *string
	var cityToUpdate *string
	var countryToUpdate *string

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

	if input.City != nil {
		if *input.City == "" {
			fieldErrors = append(fieldErrors, apperror.FieldError{Field: "city", Message: "City cannot be empty"})
		} else {
			cityToUpdate = input.City
		}
	}

	if input.Country != nil {
		if *input.Country == "" {
			fieldErrors = append(fieldErrors, apperror.FieldError{Field: "country", Message: "Country cannot be empty"})
		} else {
			countryToUpdate = input.Country
		}
	}

	if len(fieldErrors) > 0 {
		return nil,apperror.NewValidation(fieldErrors)
	}

	now := time.Now()
	if err := foundUser.Update(nameToUpdate, nicknameToUpdate, institutionToUpdate, cityToUpdate, countryToUpdate, now); err != nil {
		slog.Error("failed to apply update to user domain object", "user_id", foundUser.ID(), "error", err)
		return nil,apperror.NewInternal()
	}

	if err := uc.repo.Update(ctx, foundUser); err != nil {
		return nil,err
	}

	return &UpdateUserOutput{User: userToDTO(foundUser)}, nil
}
