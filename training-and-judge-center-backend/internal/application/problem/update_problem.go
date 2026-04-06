package problem

import (
	"context"
	"errors"
	"log/slog"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type UpdateProblemInput struct {
	Slug          string
	Title         *string
	Statement     *string
	TimeLimit     *int
	MemoryLimit   *int
	LangOverrides []LanguageOverrideInput
	Tags          []string
	Accessibility *string
	CurrentUser   user.CurrentUser
}

type UpdateProblemResult struct {
	Problem *problem.Problem
}

type UpdateProblemUseCase struct {
	repo             problem.Repository
	platformSettings problem.PlatformSettingsService
}

func NewUpdateProblemUseCase(repo problem.Repository, platformSettings problem.PlatformSettingsService) *UpdateProblemUseCase {
	return &UpdateProblemUseCase{
		repo:             repo,
		platformSettings: platformSettings,
	}
}

func (uc *UpdateProblemUseCase) Execute(ctx context.Context, input UpdateProblemInput) (*UpdateProblemResult, error) {
	slug, err := problem.NewSlug(input.Slug)
	if err != nil {
		return nil, err
	}

	p, err := uc.repo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	if p.Status.IsPublished() {
		return nil, apperror.NewBadRequest(ErrCodeProblemIsPublished, "Cannot update a published problem. Unpublish first to make changes.")
	}

	if !p.CanBeEditedBy(problem.RestoreUserID(input.CurrentUser.ID), input.CurrentUser.Role == user.RoleAdmin) {
		return nil, apperror.NewForbidden(apperror.ErrCodeForbidden, "Only the problem author, Admin, or assigned modifiers can update this problem")
	}

	var fieldErrs []apperror.FieldError

	var title *problem.Title
	if input.Title != nil {
		t, err := problem.NewTitle(*input.Title)
		if err != nil {
			var valErr *apperror.AppError
			if errors.As(err, &valErr) {
				fieldErrs = append(fieldErrs, valErr.Details...)
			} else {
				slog.ErrorContext(ctx, "failed to create title during update", "error", err, "slug", slug.String())
				return nil, apperror.NewInternal()
			}
		} else {
			title = &t
		}
	}

	globalMaxTime, globalMaxMemory := uc.platformSettings.GetGlobalLimits()

	var timeLimit **problem.TimeLimit
	if input.TimeLimit != nil {
		var tl problem.TimeLimit
		tl, err = problem.NewTimeLimit(*input.TimeLimit, globalMaxTime)
		if err != nil {
			var valErr *apperror.AppError
			if errors.As(err, &valErr) {
				fieldErrs = append(fieldErrs, valErr.Details...)
			}
		} else {
			timeLimit = new(*problem.TimeLimit)
			*timeLimit = &tl
		}
	}

	var memoryLimit **problem.MemoryLimit
	if input.MemoryLimit != nil {
		var ml problem.MemoryLimit
		ml, err = problem.NewMemoryLimit(*input.MemoryLimit, globalMaxMemory)
		if err != nil {
			var valErr *apperror.AppError
			if errors.As(err, &valErr) {
				fieldErrs = append(fieldErrs, valErr.Details...)
			}
		} else {
			memoryLimit = new(*problem.MemoryLimit)
			*memoryLimit = &ml
		}
	}

	var validOverrides []problem.LanguageOverride
	if input.LangOverrides != nil {
		seenLangs := make(map[string]struct{}, len(input.LangOverrides))
		for _, override := range input.LangOverrides {
			if override.Language == "" {
				fieldErrs = append(fieldErrs, apperror.FieldError{
					Field:   "languageOverrides",
					Message: "Language must not be empty",
				})
				continue
			}
			if !uc.platformSettings.IsLanguageSupported(override.Language) {
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

			langLimit := uc.platformSettings.GetLanguageLimit(override.Language)
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
				}
				continue
			}
			validOverrides = append(validOverrides, langOverride)
		}
	}

	var tags *problem.Tags
	if input.Tags != nil {
		allowedTags := uc.platformSettings.GetAllowedTags()
		t, err := problem.NewTags(input.Tags, allowedTags)
		if err != nil {
			var valErr *apperror.AppError
			if errors.As(err, &valErr) {
				fieldErrs = append(fieldErrs, valErr.Details...)
			}
		} else {
			tags = &t
		}
	}

	var accessibility *problem.Accessibility
	if input.Accessibility != nil {
		acc, err := problem.NewAccessibility(*input.Accessibility)
		if err != nil {
			var valErr *apperror.AppError
			if errors.As(err, &valErr) {
				fieldErrs = append(fieldErrs, valErr.Details...)
			}
		} else {
			accessibility = &acc
		}
	}

	var statementPtr *problem.Statement
	if input.Statement != nil {
		stmt, err := problem.NewStatement(input.Statement)
		if err != nil {
			var valErr *apperror.AppError
			if errors.As(err, &valErr) {
				fieldErrs = append(fieldErrs, valErr.Details...)
			} else {
				slog.ErrorContext(ctx, "failed to validate statement", "error", err)
				return nil, apperror.NewInternal()
			}
		} else {
			statementPtr = &stmt
		}
	}

	if len(fieldErrs) > 0 {
		return nil, apperror.NewValidation(fieldErrs)
	}

	p.UpdateMetadata(title, statementPtr, timeLimit, memoryLimit, validOverrides, tags)

	if accessibility != nil {
		p.UpdateAccessibility(*accessibility)
	}

	if err := uc.repo.Save(ctx, p); err != nil {
		slog.ErrorContext(ctx, "failed to save updated problem", "error", err, "slug", slug.String())
		return nil, apperror.NewInternal()
	}

	return &UpdateProblemResult{Problem: p}, nil
}
