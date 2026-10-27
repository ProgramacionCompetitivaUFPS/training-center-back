package user

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
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
	repo              user.Repository
	globalGroupJoiner GlobalGroupJoiner
	txManager         appshared.TransactionManager
}

func NewCreateUserUseCase(repo user.Repository, globalGroupJoiner GlobalGroupJoiner, txManager appshared.TransactionManager) *CreateUserUseCase {
	return &CreateUserUseCase{repo: repo, globalGroupJoiner: globalGroupJoiner, txManager: txManager}
}

type CreateUserOutput struct {
	User UserDTO
}

func (uc *CreateUserUseCase) Execute(ctx context.Context, input CreateUserInput) (*CreateUserOutput, error) {
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

	newID := uuid.New().String()
	now := time.Now()
	newUser, err := user.NewUser(newID, now, email, password, input.Name, nickname, input.Country, input.City, input.Institution)
	if err != nil {
		slog.ErrorContext(ctx, "failed to build new user domain object", "error", err)
		return nil, apperror.NewInternal()
	}

	if err := uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if err := uc.repo.Save(txCtx, newUser); err != nil {
			return err
		}
		return uc.globalGroupJoiner.AddToGlobalGroup(txCtx, newID)
	}); err != nil {
		return nil, err
	}

	return &CreateUserOutput{User: userToDTO(newUser)}, nil
}
