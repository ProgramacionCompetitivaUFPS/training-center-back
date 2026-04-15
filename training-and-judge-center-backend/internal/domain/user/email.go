package user

import (
	"fmt"
	"net/mail"
	"strings"
)

type Email struct {
	value string
}

func NewEmail(value string) (Email, error) {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return Email{}, fmt.Errorf("email is required")
	}

	parsed, err := mail.ParseAddress(trimmed)
	if err != nil {
		return Email{}, fmt.Errorf("invalid email format")
	}

	return Email{value: parsed.Address}, nil
}

func RestoreEmail(value string) Email {
	return Email{value: value}
}

func (e Email) String() string {
	return e.value
}
