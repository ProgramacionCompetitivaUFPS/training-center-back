package group

import "github.com/training-judge-center/backend/pkg/apperror"

const (
	visibilityVisible    = "VISIBLE"
	visibilityNotVisible = "NOT_VISIBLE"
)

type Visibility string

const (
	VisibilityVisible    Visibility = visibilityVisible
	VisibilityNotVisible Visibility = visibilityNotVisible
)

func NewVisibility(s string) (Visibility, error) {
	switch s {
	case visibilityVisible, visibilityNotVisible:
		return Visibility(s), nil
	}
	return "", apperror.NewValidation([]apperror.FieldError{
		{Field: "visibility", Message: "invalid visibility: " + s},
	})
}

func NewVisibilityVisible() Visibility    { return Visibility(visibilityVisible) }
func NewVisibilityNotVisible() Visibility { return Visibility(visibilityNotVisible) }

func RestoreVisibility(s string) Visibility { return Visibility(s) }
func (v Visibility) String() string         { return string(v) }
