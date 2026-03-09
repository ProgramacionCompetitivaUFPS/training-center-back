package problem

import (
	"context"

	"github.com/google/uuid"
	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/user"
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
	CurrentUser   user.CurrentUser
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
	if input.CurrentUser.Role != user.RoleCoach && input.CurrentUser.Role != user.RoleAdmin {
		return nil, apperror.NewForbidden("INSUFFICIENT_PERMISSIONS", "Only Coach and Admin users can create problems")
	}

	var fieldErrs []apperror.FieldError

	slug, err := problem.NewSlug(input.Slug)
	if err != nil {
		if valErr, ok := err.(*apperror.AppError); ok {
			fieldErrs = append(fieldErrs, valErr.Details...)
		} else {
			return nil, apperror.NewInternal()
		}
	}

	title, err := problem.NewTitle(input.Title)
	if err != nil {
		if valErr, ok := err.(*apperror.AppError); ok {
			fieldErrs = append(fieldErrs, valErr.Details...)
		} else {
			return nil, apperror.NewInternal()
		}
	}

	globalMaxTime, globalMaxMemory := usecase.platformSettings.GetGlobalLimits()

	if input.TimeLimit != nil && *input.TimeLimit > globalMaxTime {
		fieldErrs = append(fieldErrs, apperror.FieldError{
			Field:   "timeLimit",
			Message: "Exceeds global maximum time limit",
		})
	}
	if input.MemoryLimit != nil && *input.MemoryLimit > globalMaxMemory {
		fieldErrs = append(fieldErrs, apperror.FieldError{
			Field:   "memoryLimit",
			Message: "Exceeds global maximum memory limit",
		})
	}

	var validOverrides []problem.LanguageOverride
	for _, override := range input.LangOverrides {
		langOverride, err := problem.NewLanguageOverride(override.Language, override.TimeLimit, override.MemoryLimit)
		if err != nil {
			if valErr, ok := err.(*apperror.AppError); ok {
				fieldErrs = append(fieldErrs, valErr.Details...)
			} else {
				return nil, apperror.NewInternal()
			}
			continue
		}

		langMax := usecase.platformSettings.GetLanguageLimit(langOverride.Language())

		if langMax != nil {
			if langOverride.TimeLimit() != nil && *langOverride.TimeLimit() > langMax.MaxTimeLimit {
				fieldErrs = append(fieldErrs, apperror.FieldError{
					Field:   "languageOverrides.timeLimit",
					Message: "Exceeds language maximum time limit for " + langOverride.Language(),
				})
			}
			if langOverride.MemoryLimit() != nil && *langOverride.MemoryLimit() > langMax.MaxMemoryLimit {
				fieldErrs = append(fieldErrs, apperror.FieldError{
					Field:   "languageOverrides.memoryLimit",
					Message: "Exceeds language maximum memory limit for " + langOverride.Language(),
				})
			}
		} else {
			if langOverride.TimeLimit() != nil && *langOverride.TimeLimit() > globalMaxTime {
				fieldErrs = append(fieldErrs, apperror.FieldError{
					Field:   "languageOverrides.timeLimit",
					Message: "Exceeds global maximum time limit for " + langOverride.Language(),
				})
			}
			if langOverride.MemoryLimit() != nil && *langOverride.MemoryLimit() > globalMaxMemory {
				fieldErrs = append(fieldErrs, apperror.FieldError{
					Field:   "languageOverrides.memoryLimit",
					Message: "Exceeds global maximum memory limit for " + langOverride.Language(),
				})
			}
		}

		validOverrides = append(validOverrides, langOverride)
	}

	tags, err := problem.NewTags(input.Tags)
	if err != nil {
		if valErr, ok := err.(*apperror.AppError); ok {
			fieldErrs = append(fieldErrs, valErr.Details...)
		} else {
			return nil, apperror.NewInternal()
		}
	}

	for _, tag := range input.Tags {
		if !usecase.platformSettings.IsValidTag(tag) {
			fieldErrs = append(fieldErrs, apperror.FieldError{
				Field:   "tags",
				Message: "Invalid tag: " + tag,
			})
		}
	}

	if len(fieldErrs) > 0 {
		return nil, apperror.NewValidation(fieldErrs)
	}

	exists, err := usecase.repo.ExistsBySlug(ctx, slug)
	if err != nil {
		return nil, apperror.NewInternal()
	}
	if exists {
		return nil, apperror.NewConflict("SLUG_ALREADY_EXISTS", "A problem with that slug already exists")
	}

	newID := uuid.New().String()
	newProblem := problem.NewProblem(
		newID,
		slug,
		title,
		input.Statement,
		input.TimeLimit,
		input.MemoryLimit,
		validOverrides,
		tags,
		input.CurrentUser.ID,
	)

	if err := usecase.repo.Save(ctx, newProblem); err != nil {
		return nil, apperror.NewInternal()
	}

	return &CreateProblemResult{Problem: newProblem}, nil
}
