package user

import (
	"regexp"
	"strings"

	"github.com/training-judge-center/backend/pkg/apperror"
)

var validNicknameRe = regexp.MustCompile(`^[a-z0-9_-]+$`)

type Nickname struct {
	value string
}

func NewNickname(value string) (Nickname, error) {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return Nickname{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "nickname", Message: "nickname is required"},
		})
	}

	if len(trimmed) < 3 || len(trimmed) > 30 {
		return Nickname{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "nickname", Message: "nickname must be between 3 and 30 characters"},
		})
	}

	if !validNicknameRe.MatchString(trimmed) {
		return Nickname{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "nickname", Message: "nickname may only contain letters, digits, hyphens, and underscores"},
		})
	}

	return Nickname{value: trimmed}, nil
}

func RestoreNickname(value string) Nickname {
	return Nickname{value: value}
}

func (n Nickname) String() string {
	return n.value
}
