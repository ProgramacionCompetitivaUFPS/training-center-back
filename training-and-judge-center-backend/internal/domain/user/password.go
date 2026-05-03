package user

import (
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

const specialChars = "!@#$%^&*()_+-=[]{}|;:',.<>?/"

type Password struct {
	hash string
}

func NewPassword(raw string) (Password, error) {
	if len(raw) < 8 || len(raw) > 72 {
		return Password{}, fmt.Errorf("password must be between 8 and 72 characters")
	}

	var hasUpper, hasDigit, hasSpecial bool
	for _, c := range raw {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsDigit(c):
			hasDigit = true
		case strings.ContainsRune(specialChars, c):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return Password{}, fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasDigit {
		return Password{}, fmt.Errorf("password must contain at least one digit")
	}
	if !hasSpecial {
		return Password{}, fmt.Errorf("password must contain at least one special character")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return Password{}, fmt.Errorf("failed to hash password: %w", err)
	}

	return Password{hash: string(hashed)}, nil
}

const SystemNoLoginHash = "$SYSTEM_NO_LOGIN$"

func NewPasswordFromHash(hash string) (Password, error) {
	if hash == SystemNoLoginHash {
		return Password{hash: hash}, nil
	}
	if !strings.HasPrefix(hash, "$2") {
		return Password{}, fmt.Errorf("invalid bcrypt hash format")
	}
	return Password{hash: hash}, nil
}

func (p Password) Hash() string {
	return p.hash
}

func (p Password) Compare(raw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(p.hash), []byte(raw)) == nil
}
