package problem

import "github.com/training-judge-center/backend/pkg/apperror"

type LanguageLimit struct {
	language       string
	maxTimeLimit   int
	maxMemoryLimit int
}

func NewLanguageLimit(language string, maxTime, maxMemory int) (LanguageLimit, error) {
	var fieldErrs []apperror.FieldError
	if language == "" {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "language", Message: ErrCodeInvalidLanguage})
	}
	if maxTime <= 0 {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "maxTimeLimit", Message: ErrCodeInvalidTimeLimit})
	}
	if maxMemory <= 0 {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "maxMemoryLimit", Message: ErrCodeInvalidMemoryLimit})
	}
	if len(fieldErrs) > 0 {
		return LanguageLimit{}, apperror.NewValidation(fieldErrs)
	}
	return LanguageLimit{
		language:       language,
		maxTimeLimit:   maxTime,
		maxMemoryLimit: maxMemory,
	}, nil
}

func (l LanguageLimit) Language() string       { return l.language }
func (l LanguageLimit) MaxTimeLimit() int       { return l.maxTimeLimit }
func (l LanguageLimit) MaxMemoryLimit() int     { return l.maxMemoryLimit }
