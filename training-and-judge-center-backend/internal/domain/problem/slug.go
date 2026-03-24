package problem

import (
	"regexp"

	"github.com/training-judge-center/backend/pkg/apperror"
)

type Slug struct {
	value string
}

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func NewSlug(value string) (Slug, error) {
	if value == "" {
		return Slug{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "slug", Message: "Slug is required"},
		})
	}

	if len(value) < 3 {
		return Slug{}, apperror.NewBadRequest(ErrCodeSlugTooShort, "Slug must be at least 3 characters long")
	}

	if len(value) > 70 {
		return Slug{}, apperror.NewBadRequest(ErrCodeSlugTooLong, "Slug must not exceed 70 characters")
	}

	if !slugRegex.MatchString(value) {
		return Slug{}, apperror.NewBadRequest(ErrCodeSlugInvalidFormat, "Slug must contain only lowercase letters, numbers, and hyphens. Cannot start/end with hyphen or have consecutive hyphens.")
	}

	return Slug{value: value}, nil
}

func (s Slug) String() string {
	return s.value
}
