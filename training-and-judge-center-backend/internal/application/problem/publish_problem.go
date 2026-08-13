package problem

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type PublishProblemInput struct {
	Slug        string
	CurrentUser appshared.CurrentUser
}

type PublishProblemOutput struct {
	Slug           string
	MissingFields  []string
	ValidationLogs []string
	ValidationID   string
}

type PublishProblemUseCase struct {
	repo           problem.Repository
	validationRepo problem.ProblemValidationRepository
	queue          ValidationQueue
}

func NewPublishProblemUseCase(repo problem.Repository, validationRepo problem.ProblemValidationRepository, queue ValidationQueue) *PublishProblemUseCase {
	return &PublishProblemUseCase{repo: repo, validationRepo: validationRepo, queue: queue}
}

func (uc *PublishProblemUseCase) Execute(ctx context.Context, in PublishProblemInput) (*PublishProblemOutput, error) {
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
		return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions, "Only the problem author, Admin, or assigned modifiers can publish this problem")
	}

	if p.Status().IsPublished() {
		return nil, apperror.NewConflict(problem.ErrCodeAlreadyPublished, "Problem is already published")
	}

	missing, logs := requiredFieldsForPublish(p)
	if len(missing) > 0 {
		return &PublishProblemOutput{Slug: p.Slug().String(), MissingFields: missing, ValidationLogs: logs}, nil
	}

	// Reuse an in-flight validation instead of enqueueing a duplicate.
	if existing, found, err := uc.validationRepo.FindLatestByProblemID(ctx, p.ID()); err != nil {
		return nil, err
	} else if found && !existing.Status().IsFinal() {
		return &PublishProblemOutput{Slug: p.Slug().String(), ValidationID: existing.ID()}, nil
	}

	now := time.Now()
	v, err := problem.NewProblemValidation(uuid.New().String(), p.ID(), viewerID, now)
	if err != nil {
		return nil, err
	}
	if err := uc.validationRepo.Save(ctx, v); err != nil {
		if isValidationInProgress(err) {
			if existing, found, ferr := uc.validationRepo.FindLatestByProblemID(ctx, p.ID()); ferr == nil && found {
				return &PublishProblemOutput{Slug: p.Slug().String(), ValidationID: existing.ID()}, nil
			}
		}
		return nil, err
	}

	if err := uc.queue.Publish(ctx, ValidationQueueMessage{
		ValidationID: v.ID(),
		ProblemID:    p.ID(),
		Slug:         p.Slug().String(),
		Priority:     QueuePriorityPublishValidation,
		EnqueuedAt:   now,
	}); err != nil {
		return nil, err
	}

	return &PublishProblemOutput{Slug: p.Slug().String(), ValidationID: v.ID()}, nil
}

func isValidationInProgress(err error) bool {
	var appErr *apperror.AppError
	return errors.As(err, &appErr) && appErr.Code == problem.ErrCodeValidationInProgress
}

func requiredFieldsForPublish(p *problem.Problem) (missing []string, logs []string) {
	logs = append(logs, fmt.Sprintf("✓ Title: %s", p.Title().String()))

	if p.Statement().Value() == nil {
		missing = append(missing, "statement")
		logs = append(logs, "✗ Statement: Missing (required)")
	} else {
		logs = append(logs, "✓ Statement: present")
	}

	if p.TimeLimit() == nil {
		missing = append(missing, "timeLimit")
		logs = append(logs, "✗ Time limit: Missing (required)")
	} else {
		logs = append(logs, fmt.Sprintf("✓ Time limit: %dms", p.TimeLimit().Milliseconds()))
	}

	if p.MemoryLimit() == nil {
		missing = append(missing, "memoryLimit")
		logs = append(logs, "✗ Memory limit: Missing (required)")
	} else {
		logs = append(logs, fmt.Sprintf("✓ Memory limit: %d MiB", p.MemoryLimit().Megabytes()))
	}

	if p.TestCasesKey() == nil {
		missing = append(missing, "testCases")
		logs = append(logs, "✗ Test cases: Not uploaded (required)")
	} else {
		logs = append(logs, "✓ Test cases: uploaded")
	}

	if len(p.Solutions()) == 0 {
		missing = append(missing, "solution")
		logs = append(logs, "✗ Solution: Not uploaded (required)")
	} else {
		logs = append(logs, fmt.Sprintf("✓ Solution: %d uploaded", len(p.Solutions())))
	}

	return missing, logs
}
