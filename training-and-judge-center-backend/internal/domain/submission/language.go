package submission

import "github.com/training-judge-center/backend/pkg/apperror"

const (
	langCpp20     = "cpp20"
	langJava17    = "java17"
	langPython310 = "python310"
)

type Language struct {
	value string
}

func NewLanguage(value string) (Language, error) {
	switch value {
	case langCpp20, langJava17, langPython310:
		return Language{value: value}, nil
	default:
		return Language{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "language", Message: "invalid language; must be cpp20, java17, or python310"},
		})
	}
}

func RestoreLanguage(value string) Language {
	return Language{value: value}
}

func (l Language) String() string {
	return l.value
}
