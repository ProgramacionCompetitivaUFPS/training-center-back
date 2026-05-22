package problem

import "github.com/training-judge-center/backend/pkg/apperror"

const (
	accessibilityPrivate = "PRIVATE"
	accessibilityPublic  = "PUBLIC"
)

type Accessibility struct {
	value string
}

func NewAccessibility(value string) (Accessibility, error) {
	switch value {
	case accessibilityPrivate, accessibilityPublic:
		return Accessibility{value: value}, nil
	default:
		return Accessibility{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "accessibility", Message: "Invalid accessibility. Must be PRIVATE or PUBLIC"},
		})
	}
}

func NewAccessibilityPrivate() Accessibility {
	return Accessibility{value: accessibilityPrivate}
}

func NewAccessibilityPublic() Accessibility {
	return Accessibility{value: accessibilityPublic}
}

func (a Accessibility) String() string {
	return a.value
}

func RestoreAccessibility(value string) Accessibility {
	return Accessibility{value: value}
}
