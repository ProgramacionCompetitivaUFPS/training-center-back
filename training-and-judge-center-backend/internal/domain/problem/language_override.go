package problem

import "github.com/training-judge-center/backend/pkg/apperror"

type LanguageOverride struct {
	language    string
	timeLimit   *int
	memoryLimit *int
}

func NewLanguageOverride(language string, timeLimit *int, memoryLimit *int) (LanguageOverride, error) {
	var fieldErrs []apperror.FieldError

	if language == "" {
		fieldErrs = append(fieldErrs, apperror.FieldError{
			Field:   "languageOverrides.language",
			Message: "Language is required",
		})
	}

	if timeLimit != nil && *timeLimit <= 0 {
		fieldErrs = append(fieldErrs, apperror.FieldError{
			Field:   "languageOverrides.timeLimit",
			Message: "Time limit must be positive",
		})
	}

	if memoryLimit != nil && *memoryLimit <= 0 {
		fieldErrs = append(fieldErrs, apperror.FieldError{
			Field:   "languageOverrides.memoryLimit",
			Message: "Memory limit must be positive",
		})
	}

	if len(fieldErrs) > 0 {
		return LanguageOverride{}, apperror.NewValidation(fieldErrs)
	}

	return LanguageOverride{
		language:    language,
		timeLimit:   timeLimit,
		memoryLimit: memoryLimit,
	}, nil
}

func (lo LanguageOverride) Language() string {
	return lo.language
}

func (lo LanguageOverride) TimeLimit() *int {
	return lo.timeLimit
}

func (lo LanguageOverride) MemoryLimit() *int {
	return lo.memoryLimit
}
