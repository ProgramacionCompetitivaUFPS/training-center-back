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
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "language", Message: "Language is required"})
	}
	if maxTime <= 0 {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "maxTimeLimit", Message: "Max time limit must be positive"})
	}
	if maxMemory <= 0 {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "maxMemoryLimit", Message: "Max memory limit must be positive"})
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
