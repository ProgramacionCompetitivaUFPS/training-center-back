package problem

import "github.com/training-judge-center/backend/pkg/apperror"

type LanguageOverride struct {
	language    string
	timeLimit   *int
	memoryLimit *int
}

func NewLanguageOverride(
	language string,
	timeLimit *int,
	memoryLimit *int,
	langLimit *LanguageLimit,
	globalMaxTime int,
	globalMaxMemory int,
) (LanguageOverride, error) {
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

	if langLimit != nil {
		if timeLimit != nil && *timeLimit > 0 && *timeLimit > langLimit.MaxTimeLimit {
			fieldErrs = append(fieldErrs, apperror.FieldError{
				Field:   "languageOverrides.timeLimit",
				Message: "Exceeds maximum time limit for " + language,
			})
		}
		if memoryLimit != nil && *memoryLimit > 0 && *memoryLimit > langLimit.MaxMemoryLimit {
			fieldErrs = append(fieldErrs, apperror.FieldError{
				Field:   "languageOverrides.memoryLimit",
				Message: "Exceeds maximum memory limit for " + language,
			})
		}
	} else if language != "" {
		if timeLimit != nil && *timeLimit > 0 && *timeLimit > globalMaxTime {
			fieldErrs = append(fieldErrs, apperror.FieldError{
				Field:   "languageOverrides.timeLimit",
				Message: "Exceeds global maximum time limit for " + language,
			})
		}
		if memoryLimit != nil && *memoryLimit > 0 && *memoryLimit > globalMaxMemory {
			fieldErrs = append(fieldErrs, apperror.FieldError{
				Field:   "languageOverrides.memoryLimit",
				Message: "Exceeds global maximum memory limit for " + language,
			})
		}
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
