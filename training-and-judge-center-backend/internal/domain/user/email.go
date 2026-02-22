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

	if _, err := mail.ParseAddress(trimmed); err != nil {
		return Email{}, fmt.Errorf("invalid email format")
	}

	return Email{value: trimmed}, nil
}

func (e Email) String() string {
	return e.value
}
