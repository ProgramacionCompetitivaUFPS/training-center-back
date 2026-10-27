package problem

import (
	"context"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RequestProblemValidationInput struct {
	Slug        string
	CurrentUser appshared.CurrentUser
}

type RequestProblemValidationOutput struct {
	Slug           string
	MissingFields  []string
	ValidationLogs []string
	ValidationID   string
}

// RequestProblemValidationUseCase starts the same validation that publishing runs, on a draft, but
// without publishing it when it passes: the author sees whether the problem works before deciding.
// It only queues the work; the result is read through GetLatestProblemValidationUseCase.
type RequestProblemValidationUseCase struct {
	repo           problem.Repository
	validationRepo problem.ProblemValidationRepository
	queue          ValidationQueue
}

func NewRequestProblemValidationUseCase(repo problem.Repository, validationRepo problem.ProblemValidationRepository, queue ValidationQueue) *RequestProblemValidationUseCase {
	return &RequestProblemValidationUseCase{repo: repo, validationRepo: validationRepo, queue: queue}
}

func (uc *RequestProblemValidationUseCase) Execute(ctx context.Context, in RequestProblemValidationInput) (*RequestProblemValidationOutput, error) {
	slug, err := problem.NewSlug(in.Slug)
	if err != nil {
		return nil, err
	}

	p, err := uc.repo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	viewerID := shared.RestoreUserID(in.CurrentUser.ID)
	if !p.CanBeEditedBy(viewerID, in.CurrentUser.IsAdmin()) {
		return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions, "Only the problem author, Admin, or assigned modifiers can validate this problem")
	}

	if p.Status().IsPublished() {
		return nil, apperror.NewConflict(problem.ErrCodeAlreadyPublished, "Problem is already published")
	}

	missing, logs := requiredFieldsForPublish(p)
	if len(missing) > 0 {
		return &RequestProblemValidationOutput{Slug: p.Slug().String(), MissingFields: missing, ValidationLogs: logs}, nil
	}

	// A validation already in flight (this one or a publish) is reused instead of enqueueing a duplicate.
	if existing, found, err := uc.validationRepo.FindLatestByProblemID(ctx, p.ID()); err != nil {
		return nil, err
	} else if found && !existing.Status().IsFinal() {
		return &RequestProblemValidationOutput{Slug: p.Slug().String(), ValidationID: existing.ID()}, nil
	}

	id, err := enqueueValidation(ctx, uc.validationRepo, uc.queue, p, viewerID, false)
	if err != nil {
		return nil, err
	}
	return &RequestProblemValidationOutput{Slug: p.Slug().String(), ValidationID: id}, nil
}
