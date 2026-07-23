package submission

import "github.com/training-judge-center/backend/pkg/apperror"

const (
	visibilityPrivate = "PRIVATE"
	visibilityPublic  = "PUBLIC"
)

type Visibility struct{ value string }

var (
	VisibilityPrivate = Visibility{value: visibilityPrivate}
	VisibilityPublic  = Visibility{value: visibilityPublic}
)

func NewVisibility(raw string) (Visibility, error) {
	switch raw {
	case visibilityPrivate, visibilityPublic:
		return Visibility{value: raw}, nil
	default:
		return Visibility{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "visibility", Message: "must be PUBLIC or PRIVATE"},
		})
	}
}

func RestoreVisibility(raw string) Visibility {
	return Visibility{value: raw}
}

func NewVisibilityPrivate() Visibility { return Visibility{value: visibilityPrivate} }
func NewVisibilityPublic() Visibility  { return Visibility{value: visibilityPublic} }

func (v Visibility) String() string  { return v.value }
func (v Visibility) IsPublic() bool  { return v.value == visibilityPublic }
func (v Visibility) IsPrivate() bool { return v.value == visibilityPrivate }
