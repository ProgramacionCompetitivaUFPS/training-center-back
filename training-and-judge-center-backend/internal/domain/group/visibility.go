package group

import "github.com/training-judge-center/backend/pkg/apperror"

const (
	visibilityVisible    = "VISIBLE"
	visibilityNotVisible = "NOT_VISIBLE"
)

type Visibility struct{ value string }

var (
	VisibilityVisible    = Visibility{value: visibilityVisible}
	VisibilityNotVisible = Visibility{value: visibilityNotVisible}
)

func NewVisibility(s string) (Visibility, error) {
	switch s {
	case visibilityVisible, visibilityNotVisible:
		return Visibility{value: s}, nil
	}
	return Visibility{}, apperror.NewValidation([]apperror.FieldError{
		{Field: "visibility", Message: "invalid visibility: " + s},
	})
}

func NewVisibilityVisible() Visibility    { return Visibility{value: visibilityVisible} }
func NewVisibilityNotVisible() Visibility { return Visibility{value: visibilityNotVisible} }

func RestoreVisibility(s string) Visibility { return Visibility{value: s} }
func (v Visibility) String() string         { return v.value }
