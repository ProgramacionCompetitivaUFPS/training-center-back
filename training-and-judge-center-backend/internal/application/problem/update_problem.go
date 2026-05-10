package problem

import (
	"context"
	"log/slog"
	"time"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
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
	CurrentUser   appshared.CurrentUser
}

type UpdateProblemOutput struct {
	Problem ProblemDTO
}

type UpdateProblemUseCase struct {
	repo             problem.Repository
	platformSettings problem.PlatformSettings
}

func NewUpdateProblemUseCase(repo problem.Repository, platformSettings problem.PlatformSettings) *UpdateProblemUseCase {
	return &UpdateProblemUseCase{
		repo:             repo,
		platformSettings: platformSettings,
	}
}

func (uc *UpdateProblemUseCase) Execute(ctx context.Context, input UpdateProblemInput) (*UpdateProblemOutput, error) {
	slug, err := problem.NewSlug(input.Slug)
	if err != nil {
		return nil, err
	}

	p, err := uc.repo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	if p.Status().IsPublished() {
		return nil, apperror.NewBadRequest(ErrCodeProblemIsPublished, "Cannot update a published problem. Unpublish first to make changes.")
	}

	if !p.CanBeEditedBy(shared.RestoreUserID(input.CurrentUser.ID), input.CurrentUser.IsAdmin()) {
		return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions, "Only the problem author, Admin, or assigned modifiers can update this problem")
	}

	var fieldErrs []apperror.FieldError

	var title *problem.Title
	if input.Title != nil {
		t, tErr := problem.NewTitle(*input.Title)
		if err := apperror.AccumulateFieldErrors(tErr, &fieldErrs); err != nil {
			return nil, err
		}
		if tErr == nil {
			title = &t
		}
	}

	globalMaxTime, globalMaxMemory := uc.platformSettings.GlobalLimits()

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

			ll, hasLL := uc.platformSettings.LanguageLimit(override.Language)
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
	}

	var tags *problem.Tags
	if input.Tags != nil {
		allowedTags := uc.platformSettings.AllowedTags()
		t, tagsErr := problem.NewTags(input.Tags, allowedTags)
		if err := apperror.AccumulateFieldErrors(tagsErr, &fieldErrs); err != nil {
			return nil, err
		}
		if tagsErr == nil {
			tags = &t
		}
	}

	var accessibility *problem.Accessibility
	if input.Accessibility != nil {
		acc, accErr := problem.NewAccessibility(*input.Accessibility)
		if err := apperror.AccumulateFieldErrors(accErr, &fieldErrs); err != nil {
			return nil, err
		}
		if accErr == nil {
			accessibility = &acc
		}
	}

	var statementPtr *problem.Statement
	if input.Statement != nil {
		stmt, stmtErr := problem.NewStatement(input.Statement)
		if err := apperror.AccumulateFieldErrors(stmtErr, &fieldErrs); err != nil {
			return nil, err
		}
		if stmtErr == nil {
			statementPtr = &stmt
		}
	}

	if len(fieldErrs) > 0 {
		return nil, apperror.NewValidation(fieldErrs)
	}

	now := time.Now()
	p.UpdateMetadata(title, statementPtr, timeLimit, memoryLimit, validOverrides, tags, now)

	if accessibility != nil {
		p.UpdateAccessibility(*accessibility, now)
	}

	if err := uc.repo.Save(ctx, p); err != nil {
		slog.ErrorContext(ctx, "failed to save updated problem", "error", err, "slug", slug.String())
		return nil, apperror.NewInternal()
	}

	return &UpdateProblemOutput{Problem: problemToDTO(p)}, nil
}
