package user

import (
	"context"
	"log/slog"
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type AdminUpdateUserInput struct {
	TargetID    string
	Name        *string
	Nickname    *string
	Institution *string
	City        *string
	Country     *string
	Email       *string
	Role        *string
}

type AdminUpdateUserUseCase struct {
	repo user.Repository
}

func NewAdminUpdateUserUseCase(repo user.Repository) *AdminUpdateUserUseCase {
	return &AdminUpdateUserUseCase{repo: repo}
}

type AdminUpdateUserOutput struct {
	User UserDTO
}

func (uc *AdminUpdateUserUseCase) Execute(ctx context.Context, input AdminUpdateUserInput) (*AdminUpdateUserOutput, error) {
	if input.Name == nil && input.Nickname == nil && input.Institution == nil && input.City == nil && input.Country == nil && input.Email == nil && input.Role == nil {
		return nil,apperror.NewValidation([]apperror.FieldError{
			{Field: "body", Message: "At least one updatable field must be provided"},
		})
	}

	foundUser, err := uc.repo.FindByID(ctx, input.TargetID)
	if err != nil {
		slog.Error("failed to find user by id during admin update", "target_id", input.TargetID, "error", err)
		return nil,apperror.NewInternal()
	}
	if foundUser == nil {
		return nil,apperror.NewNotFound(ErrCodeUserNotFound, "User not found")
	}

	var fieldErrors []apperror.FieldError
	var nameToUpdate *string
	var nicknameToUpdate *user.Nickname
	var institutionToUpdate *string
	var cityToUpdate *string
	var countryToUpdate *string
	var emailToUpdate *user.Email
	var roleToUpdate *shared.Role

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

	if input.Email != nil {
		newEmail, err := user.NewEmail(*input.Email)
		if err != nil {
			fieldErrors = append(fieldErrors, apperror.FieldError{Field: "email", Message: err.Error()})
		} else if newEmail.String() != foundUser.Email().String() {
			emailToUpdate = &newEmail
		}
	}

	if input.Role != nil {
		newRole, err := shared.NewRole(*input.Role)
		if err != nil {
			fieldErrors = append(fieldErrors, apperror.FieldError{Field: "role", Message: "Invalid role value"})
		} else if newRole == shared.RoleAdmin {
			fieldErrors = append(fieldErrors, apperror.FieldError{Field: "role", Message: "Cannot assign ADMIN role through this endpoint"})
		} else {
			roleToUpdate = &newRole
		}
	}

	if len(fieldErrors) > 0 {
		return nil,apperror.NewValidation(fieldErrors)
	}

	now := time.Now()
	if err := foundUser.AdminUpdate(nameToUpdate, nicknameToUpdate, institutionToUpdate, cityToUpdate, countryToUpdate, emailToUpdate, roleToUpdate, now); err != nil {
		slog.Error("failed to apply admin update to user domain object", "user_id", foundUser.ID(), "error", err)
		return nil,apperror.NewInternal()
	}

	if err := uc.repo.Update(ctx, foundUser); err != nil {
		return nil,err
	}

	return &AdminUpdateUserOutput{User: userToDTO(foundUser)}, nil
}
