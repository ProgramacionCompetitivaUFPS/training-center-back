package user

import (
	"fmt"
	"regexp"
	"strings"
)

var validNicknameRe = regexp.MustCompile(`^[a-z0-9_-]+$`)

type Nickname struct {
	value string
}

func NewNickname(value string) (Nickname, error) {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return Nickname{}, fmt.Errorf("nickname is required")
	}

	if len(trimmed) < 3 || len(trimmed) > 30 {
		return Nickname{}, fmt.Errorf("nickname must be between 3 and 30 characters")
	}

	if !validNicknameRe.MatchString(trimmed) {
		return Nickname{}, fmt.Errorf("nickname may only contain letters, digits, hyphens, and underscores")
	}

	return Nickname{value: trimmed}, nil
}

func (n Nickname) String() string {
	return n.value
}
