package problem

import (
	"context"
	"time"

	"github.com/google/uuid"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type LanguageOverrideInput struct {
	Language    string
	TimeLimit   *int
	MemoryLimit *int
}

type CreateProblemInput struct {
	Slug          string
	Title         string
	Statement     *string
	TimeLimit     *int
	MemoryLimit   *int
	LangOverrides []LanguageOverrideInput
	Tags          []string
	CurrentUser   appshared.CurrentUser
}

type CreateProblemResult struct {
	Problem *problem.Problem
}

type CreateProblemUseCase struct {
	repo             problem.Repository
	platformSettings problem.PlatformSettings
}

func NewCreateProblemUseCase(repo problem.Repository, platformSettings problem.PlatformSettings) *CreateProblemUseCase {
	return &CreateProblemUseCase{
		repo:             repo,
		platformSettings: platformSettings,
	}
}

func (usecase *CreateProblemUseCase) Execute(ctx context.Context, input CreateProblemInput) (*CreateProblemResult, error) {
	if input.CurrentUser.Role != shared.RoleCoach && !input.CurrentUser.IsAdmin() {
		return nil, apperror.NewForbidden(apperror.ErrCodeForbidden, "Only Coach and Admin users can create problems")
	}

	slug, err := problem.NewSlug(input.Slug)
	if err != nil {
		return nil, err
	}

	var fieldErrs []apperror.FieldError

	title, err := problem.NewTitle(input.Title)
	if err := apperror.AccumulateFieldErrors(err, &fieldErrs); err != nil {
		return nil, err
	}

	stmt, err := problem.NewStatement(input.Statement)
	if err := apperror.AccumulateFieldErrors(err, &fieldErrs); err != nil {
		return nil, err
	}

	globalMaxTime, globalMaxMemory := usecase.platformSettings.GlobalLimits()

	var timeLimit *problem.TimeLimit
	if input.TimeLimit != nil {
		tl, tlErr := problem.NewTimeLimit(*input.TimeLimit, globalMaxTime)
		if err := apperror.AccumulateFieldErrors(tlErr, &fieldErrs); err != nil {
			return nil, err
		}
		if tlErr == nil {
			timeLimit = &tl
		}
	}

	var memoryLimit *problem.MemoryLimit
	if input.MemoryLimit != nil {
		ml, mlErr := problem.NewMemoryLimit(*input.MemoryLimit, globalMaxMemory)
		if err := apperror.AccumulateFieldErrors(mlErr, &fieldErrs); err != nil {
			return nil, err
		}
		if mlErr == nil {
			memoryLimit = &ml
		}
	}

	seenLangs := make(map[string]struct{}, len(input.LangOverrides))
	var validOverrides []problem.LanguageOverride
	for _, override := range input.LangOverrides {
		if override.Language == "" {
			fieldErrs = append(fieldErrs, apperror.FieldError{
				Field:   "languageOverrides",
				Message: "Language must not be empty",
			})
			continue
		}
		if !usecase.platformSettings.IsLanguageSupported(override.Language) {
			fieldErrs = append(fieldErrs, apperror.FieldError{
				Field:   "languageOverrides.language",
				Message: "Unsupported language: " + override.Language,
			})
			continue
		}

		if _, exists := seenLangs[override.Language]; exists {
			fieldErrs = append(fieldErrs, apperror.FieldError{
				Field:   "languageOverrides.language",
				Message: "Duplicate language override: " + override.Language,
			})
			continue
		}
		seenLangs[override.Language] = struct{}{}

		ll, hasLL := usecase.platformSettings.LanguageLimit(override.Language)
		var langLimit *problem.LanguageLimit
		if hasLL {
			langLimit = &ll
		}
		langOverride, loErr := problem.NewLanguageOverride(
			override.Language,
			override.TimeLimit,
			override.MemoryLimit,
			langLimit,
			globalMaxTime,
			globalMaxMemory,
		)
		if loErr != nil {
			if err := apperror.AccumulateFieldErrors(loErr, &fieldErrs); err != nil {
				return nil, err
			}
			continue
		}
		validOverrides = append(validOverrides, langOverride)
	}

	allowedTags := usecase.platformSettings.AllowedTags()
	tags, err := problem.NewTags(input.Tags, allowedTags)
	if err := apperror.AccumulateFieldErrors(err, &fieldErrs); err != nil {
		return nil, err
	}

	if len(fieldErrs) > 0 {
		return nil, apperror.NewValidation(fieldErrs)
	}

	newID := uuid.New().String()
	now := time.Now()
	newProblem, err := problem.NewProblem(
		newID,
		slug,
		title,
		stmt,
		timeLimit,
		memoryLimit,
		validOverrides,
		tags,
		shared.RestoreUserID(input.CurrentUser.ID),
		now,
	)
	if err != nil {
		return nil, err
	}

	if err := usecase.repo.Save(ctx, newProblem); err != nil {
		return nil, err
	}

	return &CreateProblemResult{Problem: newProblem}, nil
}
