package problem

import (
	"strings"
	"unicode"

	"github.com/training-judge-center/backend/pkg/apperror"
	"golang.org/x/text/unicode/norm"
)

type Title struct {
	value string
}

func NewTitle(value string) (Title, error) {
	if strings.TrimSpace(value) == "" {
		return Title{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "title", Message: "Title is required"},
		})
	}

	normalized := norm.NFKC.String(value)
	trimmed := strings.TrimFunc(normalized, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.Is(unicode.Cf, r)
	})

	if trimmed == "" {
		return Title{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "title", Message: "Title is required"},
		})
	}

	if len([]rune(trimmed)) > 200 {
		return Title{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "title", Message: "Title must not exceed 200 characters"},
		})
	}

	return Title{value: trimmed}, nil
}

func (t Title) String() string {
	return t.value
}

func RestoreTitle(value string) Title {
	return Title{value: value}
}
