package user

import (
	"context"
	"errors"
	"log/slog"

	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type CreateUserInput struct {
	Email       string
	Password    string
	Name        string
	Nickname    string
	Country     string
	City        string
	Institution string
}

type CreateUserUseCase struct {
	repo user.Repository
}

func NewCreateUserUseCase(repo user.Repository) *CreateUserUseCase {
	return &CreateUserUseCase{repo: repo}
}

func (uc *CreateUserUseCase) Execute(ctx context.Context, input CreateUserInput) (UserDTO, error) {
	var fieldErrors []apperror.FieldError

	email, err := user.NewEmail(input.Email)
	if err != nil {
		fieldErrors = append(fieldErrors, apperror.FieldError{Field: "email", Message: err.Error()})
	}

	password, err := user.NewPassword(input.Password)
	if err != nil {
		fieldErrors = append(fieldErrors, apperror.FieldError{Field: "password", Message: err.Error()})
	}

	nickname, err := user.NewNickname(input.Nickname)
	if err != nil {
		fieldErrors = append(fieldErrors, apperror.FieldError{Field: "nickname", Message: err.Error()})
	}

	if input.Name == "" {
		fieldErrors = append(fieldErrors, apperror.FieldError{Field: "name", Message: "Name is required"})
	}
	if input.Country == "" {
		fieldErrors = append(fieldErrors, apperror.FieldError{Field: "country", Message: "Country is required"})
	}
	if input.City == "" {
		fieldErrors = append(fieldErrors, apperror.FieldError{Field: "city", Message: "City is required"})
	}
	if input.Institution == "" {
		fieldErrors = append(fieldErrors, apperror.FieldError{Field: "institution", Message: "Institution is required"})
	}

	if len(fieldErrors) > 0 {
		return UserDTO{}, apperror.NewValidation(fieldErrors)
	}

	newUser, err := user.NewUser(email, password, input.Name, nickname, input.Country, input.City, input.Institution)
	if err != nil {
		slog.Error("failed to build new user domain object", "error", err)
		return UserDTO{}, apperror.NewInternal()
	}

	if err := uc.repo.Save(ctx, newUser); err != nil {
		if errors.Is(err, user.ErrEmailConflict) {
			return UserDTO{}, apperror.NewConflict("EMAIL_ALREADY_EXISTS", "The email address is already in use")
		}
		if errors.Is(err, user.ErrNicknameConflict) {
			return UserDTO{}, apperror.NewConflict("NICKNAME_ALREADY_EXISTS", "The nickname is already in use")
		}
		slog.Error("failed to save new user", "error", err)
		return UserDTO{}, apperror.NewInternal()
	}

	return userToDTO(newUser), nil
}
