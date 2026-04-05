package user

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type LoginInput struct {
	Email    string
	Password string
}

type LoginOutput struct {
	Token string
	User  *user.User
}

type LoginUseCase struct {
	repo         user.UserRepository
	tokenService user.TokenService
}

func NewLoginUseCase(repo user.UserRepository, tokenService user.TokenService) *LoginUseCase {
	return &LoginUseCase{repo: repo, tokenService: tokenService}
}

func (uc *LoginUseCase) Execute(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	var fieldErrors []apperror.FieldError

	if input.Email == "" {
		fieldErrors = append(fieldErrors, apperror.FieldError{Field: "email", Message: "Email is required"})
	}
	if input.Password == "" {
		fieldErrors = append(fieldErrors, apperror.FieldError{Field: "password", Message: "Password is required"})
	}
	if len(fieldErrors) > 0 {
		return nil, apperror.NewValidation(fieldErrors)
	}

	email, err := user.NewEmail(input.Email)
	if err != nil {
		return nil, apperror.NewUnauthorized("INVALID_CREDENTIALS", "Invalid email or password")
	}

	foundUser, err := uc.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, apperror.NewInternal()
	}
	if foundUser == nil {
		return nil, apperror.NewUnauthorized("INVALID_CREDENTIALS", "Invalid email or password")
	}

	if foundUser.Status() != user.StatusActive {
		return nil, apperror.NewForbidden("ACCOUNT_DEACTIVATED", "This account has been deactivated")
	}

	if !foundUser.Password().Compare(input.Password) {
		return nil, apperror.NewUnauthorized("INVALID_CREDENTIALS", "Invalid email or password")
	}

	token, err := uc.tokenService.GenerateToken(foundUser)
	if err != nil {
		return nil, apperror.NewInternal()
	}

	return &LoginOutput{Token: token, User: foundUser}, nil
}
