package material

import (
	"unicode/utf8"

	"github.com/training-judge-center/backend/pkg/apperror"
)

type Content struct {
	value string
}

func NewContent(value string) (Content, error) {
	if utf8.RuneCountInString(value) > 50000 {
		return Content{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "content", Message: "Content must not exceed 50000 characters"},
		})
	}
	return Content{value: value}, nil
}

func NewEmptyContent() Content          { return Content{value: ""} }
func RestoreContent(value string) Content { return Content{value: value} }
func (c Content) String() string          { return c.value }
