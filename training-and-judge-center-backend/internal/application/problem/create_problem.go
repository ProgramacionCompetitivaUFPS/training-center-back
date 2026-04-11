package problem

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
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
	CurrentUser   shared.CurrentUser
}

type CreateProblemResult struct {
	Problem *problem.Problem
}

type CreateProblemUseCase struct {
	repo             problem.Repository
	platformSettings problem.PlatformSettingsService
}

func NewCreateProblemUseCase(repo problem.Repository, platformSettings problem.PlatformSettingsService) *CreateProblemUseCase {
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
	if err != nil {
		var valErr *apperror.AppError
		if errors.As(err, &valErr) {
			fieldErrs = append(fieldErrs, valErr.Details...)
		} else {
			return nil, apperror.NewInternal()
		}
	}

	stmt, err := problem.NewStatement(input.Statement)
	if err != nil {
		var valErr *apperror.AppError
		if errors.As(err, &valErr) {
			fieldErrs = append(fieldErrs, valErr.Details...)
		} else {
			return nil, apperror.NewInternal()
		}
	}

	globalMaxTime, globalMaxMemory := usecase.platformSettings.GetGlobalLimits()

	var timeLimit *problem.TimeLimit
	if input.TimeLimit != nil {
		tl, err := problem.NewTimeLimit(*input.TimeLimit, globalMaxTime)
		if err != nil {
			var valErr *apperror.AppError
			if errors.As(err, &valErr) {
				fieldErrs = append(fieldErrs, valErr.Details...)
			} else {
				return nil, apperror.NewInternal()
			}
		} else {
			timeLimit = &tl
		}
	}

	var memoryLimit *problem.MemoryLimit
	if input.MemoryLimit != nil {
		ml, err := problem.NewMemoryLimit(*input.MemoryLimit, globalMaxMemory)
		if err != nil {
			var valErr *apperror.AppError
			if errors.As(err, &valErr) {
				fieldErrs = append(fieldErrs, valErr.Details...)
			} else {
				return nil, apperror.NewInternal()
			}
		} else {
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

		langLimit := usecase.platformSettings.GetLanguageLimit(override.Language)
		langOverride, err := problem.NewLanguageOverride(
			override.Language,
			override.TimeLimit,
			override.MemoryLimit,
			langLimit,
			globalMaxTime,
			globalMaxMemory,
		)
		if err != nil {
			var valErr *apperror.AppError
			if errors.As(err, &valErr) {
				fieldErrs = append(fieldErrs, valErr.Details...)
			} else {
				return nil, apperror.NewInternal()
			}
			continue
		}
		validOverrides = append(validOverrides, langOverride)
	}

	allowedTags := usecase.platformSettings.GetAllowedTags()
	tags, err := problem.NewTags(input.Tags, allowedTags)
	if err != nil {
		var valErr *apperror.AppError
		if errors.As(err, &valErr) {
			fieldErrs = append(fieldErrs, valErr.Details...)
		} else {
			return nil, apperror.NewInternal()
		}
	}

	if len(fieldErrs) > 0 {
		return nil, apperror.NewValidation(fieldErrs)
	}

	newID := uuid.New().String()
	newProblem := problem.NewProblem(
		newID,
		slug,
		title,
		stmt,
		timeLimit,
		memoryLimit,
		validOverrides,
		tags,
		shared.RestoreUserID(input.CurrentUser.ID),
	)

	if err := usecase.repo.Save(ctx, newProblem); err != nil {
		var slugErr *problem.ErrSlugAlreadyExists
		if errors.As(err, &slugErr) {
			return nil, apperror.NewConflict(ErrCodeProblemSlugAlreadyExists, "A problem with slug '"+slugErr.Slug+"' already exists")
		}
		slog.ErrorContext(ctx, "failed to save new problem", "error", err, "slug", slug.String())
		return nil, apperror.NewInternal()
	}

	return &CreateProblemResult{Problem: newProblem}, nil
}
