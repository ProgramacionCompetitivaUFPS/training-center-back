package user

import (
	"context"

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
	repo user.UserRepository
}

func NewCreateUserUseCase(repo user.UserRepository) *CreateUserUseCase {
	return &CreateUserUseCase{repo: repo}
}

func (uc *CreateUserUseCase) Execute(ctx context.Context, input CreateUserInput) (*user.User, error) {
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
		return nil, apperror.NewValidation(fieldErrors)
	}

	emailExists, err := uc.repo.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, apperror.NewInternal()
	}
	if emailExists {
		return nil, apperror.NewConflict("EMAIL_ALREADY_EXISTS", "The email address is already in use")
	}

	nicknameExists, err := uc.repo.ExistsByNickname(ctx, nickname)
	if err != nil {
		return nil, apperror.NewInternal()
	}
	if nicknameExists {
		return nil, apperror.NewConflict("NICKNAME_ALREADY_EXISTS", "The nickname is already in use")
	}

	newUser := user.NewUser(email, password, input.Name, nickname, input.Country, input.City, input.Institution)

	if err := uc.repo.Save(ctx, newUser); err != nil {
		return nil, apperror.NewInternal()
	}

	return newUser, nil
}
