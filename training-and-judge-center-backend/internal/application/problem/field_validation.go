package problem

import (
	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func validated[T any](fieldErrs *[]apperror.FieldError, value T, err error) (*T, error) {
	if accErr := apperror.AccumulateFieldErrors(err, fieldErrs); accErr != nil {
		return nil, accErr
	}
	if err != nil {
		return nil, nil
	}
	return &value, nil
}

func validateIfSet[In, T any](fieldErrs *[]apperror.FieldError, in *In, build func(In) (T, error)) (*T, error) {
	if in == nil {
		return nil, nil
	}
	value, err := build(*in)
	return validated(fieldErrs, value, err)
}

func validateLimits(
	settings problem.PlatformSettings,
	timeLimit, memoryLimit *int,
	fieldErrs *[]apperror.FieldError,
) (*problem.TimeLimit, *problem.MemoryLimit, error) {
	globalMaxTime, globalMaxMemory := settings.GlobalLimits()

	validTime, err := validateIfSet(fieldErrs, timeLimit, func(v int) (problem.TimeLimit, error) {
		return problem.NewTimeLimit(v, globalMaxTime)
	})
	if err != nil {
		return nil, nil, err
	}

	validMemory, err := validateIfSet(fieldErrs, memoryLimit, func(v int) (problem.MemoryLimit, error) {
		return problem.NewMemoryLimit(v, globalMaxMemory)
	})
	if err != nil {
		return nil, nil, err
	}

	return validTime, validMemory, nil
}

func validateLanguageOverrides(
	settings problem.PlatformSettings,
	inputs []LanguageOverrideInput,
	fieldErrs *[]apperror.FieldError,
) ([]problem.LanguageOverride, error) {
	globalMaxTime, globalMaxMemory := settings.GlobalLimits()
	seenLangs := make(map[string]struct{}, len(inputs))
	var valid []problem.LanguageOverride

	for _, override := range inputs {
		if fieldErr := checkOverrideLanguage(settings, override.Language, seenLangs); fieldErr != nil {
			*fieldErrs = append(*fieldErrs, *fieldErr)
			continue
		}
		seenLangs[override.Language] = struct{}{}

		built, err := buildLanguageOverride(settings, override, globalMaxTime, globalMaxMemory)
		if err != nil {
			if accErr := apperror.AccumulateFieldErrors(err, fieldErrs); accErr != nil {
				return nil, accErr
			}
			continue
		}
		valid = append(valid, built)
	}

	return valid, nil
}

func checkOverrideLanguage(settings problem.PlatformSettings, language string, seen map[string]struct{}) *apperror.FieldError {
	if language == "" {
		return &apperror.FieldError{Field: "languageOverrides", Message: "Language must not be empty"}
	}
	if !settings.IsLanguageSupported(language) {
		return &apperror.FieldError{Field: "languageOverrides.language", Message: "Unsupported language: " + language}
	}
	if _, exists := seen[language]; exists {
		return &apperror.FieldError{Field: "languageOverrides.language", Message: "Duplicate language override: " + language}
	}
	return nil
}

func buildLanguageOverride(
	settings problem.PlatformSettings,
	override LanguageOverrideInput,
	globalMaxTime, globalMaxMemory int,
) (problem.LanguageOverride, error) {
	var langLimit *problem.LanguageLimit
	if limit, ok := settings.LanguageLimit(override.Language); ok {
		langLimit = &limit
	}
	return problem.NewLanguageOverride(
		override.Language,
		override.TimeLimit,
		override.MemoryLimit,
		langLimit,
		globalMaxTime,
		globalMaxMemory,
	)
}
