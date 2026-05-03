package user

import (
	"net/mail"
	"strings"

	"github.com/training-judge-center/backend/pkg/apperror"
)

type Email struct {
	value string
}

func NewEmail(value string) (Email, error) {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return Email{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "email", Message: "email is required"},
		})
	}

	parsed, err := mail.ParseAddress(trimmed)
	if err != nil {
		return Email{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "email", Message: "invalid email format"},
		})
	}

	return Email{value: parsed.Address}, nil
}

func RestoreEmail(value string) Email {
	return Email{value: value}
}

func (e Email) String() string {
	return e.value
}
