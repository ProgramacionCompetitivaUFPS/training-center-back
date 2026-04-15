package problem

import "github.com/training-judge-center/backend/pkg/apperror"

type Accessibility struct {
	value string
}

func NewAccessibility(value string) (Accessibility, error) {
	switch value {
	case "PRIVATE", "PUBLIC":
		return Accessibility{value: value}, nil
	default:
		return Accessibility{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "accessibility", Message: "Invalid accessibility. Must be PRIVATE or PUBLIC"},
		})
	}
}

func NewAccessibilityPrivate() Accessibility {
	return Accessibility{value: "PRIVATE"}
}

func NewAccessibilityPublic() Accessibility {
	return Accessibility{value: "PUBLIC"}
}

func (a Accessibility) String() string {
	return a.value
}

func RestoreAccessibility(value string) Accessibility {
	return Accessibility{value: value}
}
