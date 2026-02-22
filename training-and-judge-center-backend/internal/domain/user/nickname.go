package user

import (
	"fmt"
	"strings"
)

type Nickname struct {
	value string
}

func NewNickname(value string) (Nickname, error) {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return Nickname{}, fmt.Errorf("nickname is required")
	}

	if len(trimmed) < 3 || len(trimmed) > 20 {
		return Nickname{}, fmt.Errorf("nickname must be between 3 and 20 characters")
	}

	return Nickname{value: trimmed}, nil
}

func (n Nickname) String() string {
	return n.value
}
