package problem

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
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
	validationRepo   problem.ProblemValidationRepository
	platformSettings problem.PlatformSettings
}

type updateFields struct {
	title         *problem.Title
	statement     *problem.Statement
	timeLimit     *problem.TimeLimit
	memoryLimit   *problem.MemoryLimit
	overrides     []problem.LanguageOverride
	tags          *problem.Tags
	accessibility *problem.Accessibility
}

func NewUpdateProblemUseCase(repo problem.Repository, validationRepo problem.ProblemValidationRepository, platformSettings problem.PlatformSettings) *UpdateProblemUseCase {
	return &UpdateProblemUseCase{
		repo:             repo,
		validationRepo:   validationRepo,
		platformSettings: platformSettings,
	}
}

func (uc *UpdateProblemUseCase) Execute(ctx context.Context, input UpdateProblemInput) (*UpdateProblemOutput, error) {
	p, err := uc.loadEditableProblem(ctx, input)
	if err != nil {
		return nil, err
	}

	fields, err := uc.validateFields(input)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	p.UpdateMetadata(fields.title, fields.statement, fields.timeLimit, fields.memoryLimit, fields.overrides, fields.tags, now)

	if fields.accessibility != nil {
		p.UpdateAccessibility(*fields.accessibility, now)
	}

	if err := uc.repo.Save(ctx, p); err != nil {
		return nil, err
	}

	return &UpdateProblemOutput{Problem: problemToDTO(p)}, nil
}

func (uc *UpdateProblemUseCase) loadEditableProblem(ctx context.Context, input UpdateProblemInput) (*problem.Problem, error) {
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

	if err := ensureNoActiveValidation(ctx, uc.validationRepo, p.ID()); err != nil {
		return nil, err
	}

	return p, nil
}

func (uc *UpdateProblemUseCase) validateFields(input UpdateProblemInput) (*updateFields, error) {
	var fieldErrs []apperror.FieldError
	fields := &updateFields{}
	var err error

	if fields.title, err = validateIfSet(&fieldErrs, input.Title, problem.NewTitle); err != nil {
		return nil, err
	}

	if fields.timeLimit, fields.memoryLimit, err = validateLimits(uc.platformSettings, input.TimeLimit, input.MemoryLimit, &fieldErrs); err != nil {
		return nil, err
	}

	if fields.overrides, err = validateLanguageOverrides(uc.platformSettings, input.LangOverrides, &fieldErrs); err != nil {
		return nil, err
	}

	if input.Tags != nil {
		allowedTags := uc.platformSettings.AllowedTags()
		fields.tags, err = validateIfSet(&fieldErrs, &input.Tags, func(values []string) (problem.Tags, error) {
			return problem.NewTags(values, allowedTags)
		})
		if err != nil {
			return nil, err
		}
	}

	if fields.accessibility, err = validateIfSet(&fieldErrs, input.Accessibility, problem.NewAccessibility); err != nil {
		return nil, err
	}

	if input.Statement != nil {
		statement, statementErr := problem.NewStatement(input.Statement)
		if fields.statement, err = validated(&fieldErrs, statement, statementErr); err != nil {
			return nil, err
		}
	}

	if len(fieldErrs) > 0 {
		return nil, apperror.NewValidation(fieldErrs)
	}

	return fields, nil
}
