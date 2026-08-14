package user

import (
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"

	"github.com/training-judge-center/backend/pkg/apperror"
)

const specialChars = "!@#$%^&*()_+-=[]{}|;:',.<>?/"

type Password struct {
	hash string
}

func NewPassword(raw string) (Password, error) {
	if len(raw) < 8 || len(raw) > 72 {
		return Password{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "password", Message: "password must be between 8 and 72 characters"},
		})
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
		return Password{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "password", Message: "password must contain at least one uppercase letter"},
		})
	}
	if !hasDigit {
		return Password{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "password", Message: "password must contain at least one digit"},
		})
	}
	if !hasSpecial {
		return Password{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "password", Message: "password must contain at least one special character"},
		})
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return Password{}, apperror.NewInternal()
	}

	return Password{hash: string(hashed)}, nil
}

const SystemNoLoginHash = "$SYSTEM_NO_LOGIN$"

// NoPassword represents a user with no local password — e.g. one authenticated
// only through an external provider such as Google.
func NoPassword() Password {
	return Password{}
}

func (p Password) HasPassword() bool {
	return p.hash != ""
}

func RestorePassword(hash string) Password {
	return Password{hash: hash}
}

func (p Password) Hash() string {
	return p.hash
}

func (p Password) Compare(raw string) bool {
	if !p.HasPassword() {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(p.hash), []byte(raw)) == nil
}
