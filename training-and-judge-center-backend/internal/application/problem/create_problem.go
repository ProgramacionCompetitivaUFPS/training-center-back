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

type CreateProblemOutput struct {
	Problem ProblemDTO
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

func (usecase *CreateProblemUseCase) Execute(ctx context.Context, input CreateProblemInput) (*CreateProblemOutput, error) {
	if input.CurrentUser.Role != shared.RoleCoach && !input.CurrentUser.IsAdmin() {
		return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions, "Only Coach and Admin users can create problems")
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

	timeLimit, memoryLimit, err := validateLimits(usecase.platformSettings, input.TimeLimit, input.MemoryLimit, &fieldErrs)
	if err != nil {
		return nil, err
	}

	validOverrides, err := validateLanguageOverrides(usecase.platformSettings, input.LangOverrides, &fieldErrs)
	if err != nil {
		return nil, err
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

	return &CreateProblemOutput{Problem: problemToDTO(newProblem)}, nil
}
